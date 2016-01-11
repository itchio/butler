package main

import (
	"archive/tar"
	"io"
	"os"
	"path"
)

const MODE_MASK = 0666

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
		err = os.MkdirAll(dir, 0755)
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
	err := os.MkdirAll(filename, mode)
	if err != nil {
		Die(err.Error())
	}
}

func untarReg(filename string, mode os.FileMode, tarReader io.Reader) {
	writer, err := os.Create(filename)

	if err != nil {
		Die(err.Error())
	}
	defer writer.Close()

	io.Copy(writer, tarReader)

	err = os.Chmod(filename, os.FileMode(mode))
	if err != nil {
		Die(err.Error())
	}
}

func untarSymlink(linkname string, filename string) {
	_, err := os.Lstat(filename)
	if err == nil {
		// already exists, overwriting
		err = os.Remove(filename)
	}

	err = os.Symlink(linkname, filename)
	if err != nil {
		Die(err.Error())
	}
}
