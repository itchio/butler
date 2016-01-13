package main

import (
	"archive/tar"
	"io"
	"os"
	"path"
)

const MODE_MASK = 0666
const DIR_MODE = 0777

func untar(archive string, dir string) {
	Logf("extracting %s to %s", archive, dir)

	dirCount := 0
	regCount := 0
	symlinkCount := 0

	file, err := os.Open(archive)

	if err != nil {
		Die(err.Error())
	}

	defer file.Close()

	_, err = os.Lstat(dir)
	if err != nil {
		Logf("destination %s does not exist, creating...", dir)
		err = os.MkdirAll(dir, DIR_MODE)
	}

	tarReader := tar.NewReader(file)

	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			Die(err.Error())
		}

		rel_filename := header.Name
		filename := path.Join(dir, rel_filename)

		switch header.Typeflag {
		case tar.TypeDir:
			untarDir(filename, os.FileMode(header.Mode|MODE_MASK))
			dirCount += 1

		case tar.TypeReg:
			untarReg(filename, os.FileMode(header.Mode|MODE_MASK), tarReader)
			regCount += 1

		case tar.TypeSymlink:
			untarSymlink(header.Linkname, filename)
			symlinkCount += 1

		default:
			Dief("Unable to untar entry of type %d", header.Typeflag)
		}
	}
	Logf("extracted %d dirs, %d files, %d symlinks", dirCount, regCount, symlinkCount)
}

func untarDir(filename string, mode os.FileMode) {
	must(os.MkdirAll(filename, mode))
}

func untarReg(filename string, mode os.FileMode, tarReader io.Reader) {
	must(os.RemoveAll(filename))

	writer, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	must(err)
	defer writer.Close()

	_, err = io.Copy(writer, tarReader)
	must(err)
}

func untarSymlink(linkname string, filename string) {
	must(os.RemoveAll(filename))
	must(os.Symlink(linkname, filename))
}
