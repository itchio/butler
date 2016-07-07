package archiver

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/sync"
	"github.com/itchio/wharf/tlc"
)

func ExtractZip(readerAt io.ReaderAt, size int64, dir string, consumer *pwr.StateConsumer) (*ExtractResult, error) {
	consumer.Infof("Extracting a zip archive to %s", dir)

	dirCount := 0
	regCount := 0
	symlinkCount := 0

	reader, err := zip.NewReader(readerAt, size)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	for _, file := range reader.File {
		err = func() error {
			rel := file.Name
			filename := path.Join(dir, filepath.FromSlash(rel))

			info := file.FileInfo()
			mode := info.Mode()

			fileReader, err := file.Open()
			if err != nil {
				return errors.Wrap(err, 1)
			}
			defer fileReader.Close()

			if info.IsDir() {
				err = Mkdir(filename)
				if err != nil {
					return errors.Wrap(err, 1)
				}
				dirCount++
			} else if mode&os.ModeSymlink > 0 {
				linkname, err := ioutil.ReadAll(fileReader)
				err = Symlink(string(linkname), filename, consumer)
				if err != nil {
					return errors.Wrap(err, 1)
				}
				symlinkCount++
			} else {
				err = CopyFile(filename, os.FileMode(mode&LuckyMode|ModeMask), fileReader, consumer)
				if err != nil {
					return errors.Wrap(err, 1)
				}
				regCount++
			}

			return nil
		}()
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}
	}

	return &ExtractResult{
		Dirs:     dirCount,
		Files:    regCount,
		Symlinks: symlinkCount,
	}, nil
}

func CompressZip(archiveWriter io.Writer, container *tlc.Container, pool sync.FilePool, consumer *pwr.StateConsumer) (*CompressResult, error) {
	var err error
	var uncompressedSize int64
	var compressedSize int64

	archiveCounter := counter.NewWriter(archiveWriter)

	zipWriter := zip.NewWriter(archiveCounter)
	defer zipWriter.Close()
	defer func() {
		if zipWriter != nil {
			if zErr := zipWriter.Close(); err == nil && zErr != nil {
				err = errors.Wrap(zErr, 1)
			}
		}
	}()

	for _, dir := range container.Dirs {
		fh := zip.FileHeader{
			Name: dir.Path + "/",
		}
		fh.SetMode(os.FileMode(dir.Mode))

		_, err := zipWriter.CreateHeader(&fh)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}
	}

	for fileIndex, file := range container.Files {
		fh := zip.FileHeader{
			Name:               file.Path,
			UncompressedSize64: uint64(file.Size),
			Method:             zip.Deflate,
		}
		fh.SetMode(os.FileMode(file.Mode))

		entryWriter, err := zipWriter.CreateHeader(&fh)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		entryReader, err := pool.GetReader(int64(fileIndex))
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		copiedBytes, err := io.Copy(entryWriter, entryReader)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		uncompressedSize += copiedBytes
	}

	for _, symlink := range container.Symlinks {
		fh := zip.FileHeader{
			Name: symlink.Path,
		}
		fh.SetMode(os.FileMode(symlink.Mode))

		entryWriter, err := zipWriter.CreateHeader(&fh)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		entryWriter.Write([]byte(symlink.Dest))
	}

	err = zipWriter.Close()
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	zipWriter = nil

	compressedSize = archiveCounter.Count()

	return &CompressResult{
		UncompressedSize: uncompressedSize,
		CompressedSize:   compressedSize,
	}, nil
}
