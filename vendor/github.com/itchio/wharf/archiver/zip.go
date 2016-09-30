package archiver

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

func ExtractZip(readerAt io.ReaderAt, size int64, dir string, consumer *pwr.StateConsumer) (*ExtractResult, error) {
	dirCount := 0
	regCount := 0
	symlinkCount := 0

	reader, err := zip.NewReader(readerAt, size)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	totalSize := uint64(0)
	for _, file := range reader.File {
		totalSize += file.UncompressedSize64
	}

	doneSize := uint64(0)

	for _, file := range reader.File {
		err = func() error {
			rel := file.Name
			filename := path.Join(dir, filepath.FromSlash(rel))

			info := file.FileInfo()
			mode := info.Mode()

			fileReader, fErr := file.Open()
			if fErr != nil {
				return errors.Wrap(fErr, 1)
			}
			defer fileReader.Close()

			if info.IsDir() {
				err = Mkdir(filename)
				if err != nil {
					return errors.Wrap(err, 1)
				}
				dirCount++
			} else if mode&os.ModeSymlink > 0 {
				linkname, lErr := ioutil.ReadAll(fileReader)
				lErr = Symlink(string(linkname), filename, consumer)
				if lErr != nil {
					return errors.Wrap(err, 1)
				}
				symlinkCount++
			} else {
				countingReader := counter.NewReaderCallback(func(offset int64) {
					currentSize := int64(doneSize) + offset
					consumer.Progress(float64(currentSize) / float64(totalSize))
				}, fileReader)

				err = CopyFile(filename, os.FileMode(mode&LuckyMode|ModeMask), countingReader, consumer)
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

		doneSize += file.UncompressedSize64
		consumer.Progress(float64(doneSize) / float64(totalSize))
	}

	return &ExtractResult{
		Dirs:     dirCount,
		Files:    regCount,
		Symlinks: symlinkCount,
	}, nil
}

func CompressZip(archiveWriter io.Writer, container *tlc.Container, pool wsync.Pool, consumer *pwr.StateConsumer) (*CompressResult, error) {
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
		fh.SetModTime(time.Now())

		_, hErr := zipWriter.CreateHeader(&fh)
		if hErr != nil {
			return nil, errors.Wrap(hErr, 1)
		}
	}

	for fileIndex, file := range container.Files {
		fh := zip.FileHeader{
			Name:               file.Path,
			UncompressedSize64: uint64(file.Size),
			Method:             zip.Deflate,
		}
		fh.SetMode(os.FileMode(file.Mode))
		fh.SetModTime(time.Now())

		entryWriter, eErr := zipWriter.CreateHeader(&fh)
		if eErr != nil {
			return nil, errors.Wrap(eErr, 1)
		}

		entryReader, eErr := pool.GetReader(int64(fileIndex))
		if eErr != nil {
			return nil, errors.Wrap(eErr, 1)
		}

		copiedBytes, eErr := io.Copy(entryWriter, entryReader)
		if eErr != nil {
			return nil, errors.Wrap(eErr, 1)
		}

		uncompressedSize += copiedBytes
	}

	for _, symlink := range container.Symlinks {
		fh := zip.FileHeader{
			Name: symlink.Path,
		}
		fh.SetMode(os.FileMode(symlink.Mode))

		entryWriter, eErr := zipWriter.CreateHeader(&fh)
		if eErr != nil {
			return nil, errors.Wrap(eErr, 1)
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
