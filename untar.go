package main

import (
	"archive/tar"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/itchio/butler/comm"
)

// Does not preserve users, nor permission, except the executable bit
func untar(archive string, dir string) {
	comm.Logf("Extracting %s to %s", archive, dir)

	dirCount := 0
	regCount := 0
	symlinkCount := 0

	file, err := os.Open(archive)
	must(err)

	defer file.Close()

	dittoMkdir(dir)

	tarReader := tar.NewReader(file)

	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			must(err)
		}

		rel := header.Name
		filename := path.Join(dir, rel)

		switch header.Typeflag {
		case tar.TypeDir:
			dittoMkdir(filename)
			dirCount += 1

		case tar.TypeReg:
			untarReg(filename, os.FileMode(header.Mode&LUCKY_MODE|MODE_MASK), tarReader)
			regCount += 1

		case tar.TypeSymlink:
			untarSymlink(header.Linkname, filename)
			symlinkCount += 1

		default:
			comm.Dief("Unable to untar entry of type %d", header.Typeflag)
		}
	}

	comm.Logf("Extracted %d dirs, %d files, %d symlinks", dirCount, regCount, symlinkCount)
}

func untarReg(filename string, mode os.FileMode, tarReader io.Reader) {
	comm.Debugf("extract %s", filename)
	must(os.RemoveAll(filename))

	dirname := filepath.Dir(filename)
	must(os.MkdirAll(dirname, LUCKY_MODE))

	writer, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	must(err)
	defer writer.Close()

	_, err = io.Copy(writer, tarReader)
	must(err)

	must(os.Chmod(filename, mode))
}

func untarSymlink(linkname string, filename string) {
	comm.Debugf("ln -s %s %s", linkname, filename)

	must(os.RemoveAll(filename))
	must(os.Symlink(linkname, filename))
}
