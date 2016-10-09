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
	"github.com/itchio/wharf/state"
)

func ExtractZip(readerAt io.ReaderAt, size int64, dir string, consumer *state.Consumer) (*ExtractResult, error) {
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

func CompressZip(archiveWriter io.Writer, dir string, consumer *state.Consumer) (*CompressResult, error) {
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

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		name, wErr := filepath.Rel(dir, path)
		if wErr != nil {
			return wErr
		}

		if name == "." {
			// don't add '.' to zip
			return nil
		}

		name = filepath.ToSlash(name)

		fh, wErr := zip.FileInfoHeader(info)
		if wErr != nil {
			return wErr
		}

		fh.Name = name

		writer, wErr := zipWriter.CreateHeader(fh)
		if wErr != nil {
			return wErr
		}

		if info.IsDir() {
			// good!
		} else if info.Mode()&os.ModeSymlink > 0 {
			dest, wErr := os.Readlink(path)
			if wErr != nil {
				return wErr
			}

			_, wErr = writer.Write([]byte(dest))
			if wErr != nil {
				return wErr
			}
		} else if info.Mode().IsRegular() {
			reader, wErr := os.Open(path)
			if wErr != nil {
				return wErr
			}
			defer reader.Close()

			copiedBytes, wErr := io.Copy(writer, reader)
			if wErr != nil {
				return wErr
			}

			uncompressedSize += copiedBytes
		}

		return nil
	})

	err = zipWriter.Close()
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	zipWriter = nil

	compressedSize = archiveCounter.Count()

	return &CompressResult{
		UncompressedSize: uncompressedSize,
		CompressedSize:   compressedSize,
	}, err
}
