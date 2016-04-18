package archiver

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/itchio/wharf/pwr"
)

// Does not preserve users, nor permission, except the executable bit
func ExtractTar(archive string, dir string, consumer *pwr.StateConsumer) (*ExtractResult, error) {
	consumer.Infof("Extracting %s to %s", archive, dir)

	dirCount := 0
	regCount := 0
	symlinkCount := 0

	file, err := os.Open(archive)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	err = Mkdir(dir)
	if err != nil {
		return nil, err
	}

	tarReader := tar.NewReader(file)

	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		rel := header.Name
		filename := path.Join(dir, filepath.FromSlash(rel))

		switch header.Typeflag {
		case tar.TypeDir:
			err = Mkdir(filename)
			if err != nil {
				return nil, err
			}
			dirCount++

		case tar.TypeReg:
			err = CopyFile(filename, os.FileMode(header.Mode&LuckyMode|ModeMask), tarReader, consumer)
			if err != nil {
				return nil, err
			}
			regCount++

		case tar.TypeSymlink:
			err = Symlink(header.Linkname, filename, consumer)
			if err != nil {
				return nil, err
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
