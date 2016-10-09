package archiver

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

// Does not preserve users, nor permission, except the executable bit
func ExtractTar(archive string, dir string, settings ExtractSettings) (*ExtractResult, error) {
	settings.Consumer.Infof("Extracting %s to %s", archive, dir)

	dirCount := 0
	regCount := 0
	symlinkCount := 0

	file, err := eos.Open(archive)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	defer file.Close()

	err = Mkdir(dir)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	tarReader := tar.NewReader(file)

	for {
		header, err := tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, errors.Wrap(err, 1)
		}

		rel := header.Name
		filename := path.Join(dir, filepath.FromSlash(rel))

		switch header.Typeflag {
		case tar.TypeDir:
			err = Mkdir(filename)
			if err != nil {
				return nil, errors.Wrap(err, 1)
			}
			dirCount++

		case tar.TypeReg:
			settings.Consumer.Debugf("extract %s", filename)
			err = CopyFile(filename, os.FileMode(header.Mode&LuckyMode|ModeMask), tarReader)
			if err != nil {
				return nil, errors.Wrap(err, 1)
			}
			regCount++

		case tar.TypeSymlink:
			err = Symlink(header.Linkname, filename, settings.Consumer)
			if err != nil {
				return nil, errors.Wrap(err, 1)
			}
			symlinkCount++

		default:
			return nil, fmt.Errorf("Unable to untar entry of type %d", header.Typeflag)
		}
	}

	return &ExtractResult{
		Dirs:     dirCount,
		Files:    regCount,
		Symlinks: symlinkCount,
	}, nil
}

func CompressTar(archiveWriter io.Writer, dir string, consumer *state.Consumer) (*CompressResult, error) {
	var err error
	var uncompressedSize int64
	var compressedSize int64

	archiveCounter := counter.NewWriter(archiveWriter)

	tarWriter := tar.NewWriter(archiveCounter)
	defer tarWriter.Close()
	defer func() {
		if tarWriter != nil {
			if zErr := tarWriter.Close(); err == nil && zErr != nil {
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

		th, wErr := tar.FileInfoHeader(info, "")
		if wErr != nil {
			return wErr
		}

		th.Name = name

		if info.IsDir() {
			// good!
			wErr = tarWriter.WriteHeader(th)
			if wErr != nil {
				return wErr
			}
		} else if info.Mode()&os.ModeSymlink > 0 {
			dest, lErr := os.Readlink(path)
			if lErr != nil {
				return lErr
			}

			th.Linkname = dest

			lErr = tarWriter.WriteHeader(th)
			if lErr != nil {
				return lErr
			}
		} else if info.Mode().IsRegular() {
			wErr = tarWriter.WriteHeader(th)
			if wErr != nil {
				return wErr
			}

			reader, wErr := os.Open(path)
			if wErr != nil {
				return wErr
			}
			defer reader.Close()

			copiedBytes, wErr := io.Copy(tarWriter, reader)
			if wErr != nil {
				return wErr
			}

			uncompressedSize += copiedBytes
		}

		return nil
	})

	err = tarWriter.Close()
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	tarWriter = nil

	compressedSize = archiveCounter.Count()

	return &CompressResult{
		UncompressedSize: uncompressedSize,
		CompressedSize:   compressedSize,
	}, err
}
