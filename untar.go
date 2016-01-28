package main

import (
	"archive/tar"
	"io"
	"os"
	"path"
	"path/filepath"
)

// Does not preserve users, nor permission, except the executable bit
func untar(archive string, dir string) {
	if !*appArgs.quiet {
		Logf("extracting %s to %s", archive, dir)
	}

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
			Die(err.Error())
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
			Dief("Unable to untar entry of type %d", header.Typeflag)
		}
	}

	if !*appArgs.quiet {
		Logf("extracted %d dirs, %d files, %d symlinks", dirCount, regCount, symlinkCount)
	}
}

func untarReg(filename string, mode os.FileMode, tarReader io.Reader) {
	if *appArgs.verbose {
		Logf("extract %s", filename)
	}
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
	if *appArgs.verbose {
		Logf("ln -s %s %s", linkname, filename)
	}

	must(os.RemoveAll(filename))
	must(os.Symlink(linkname, filename))
}
