package main

import (
	"io"
	"os"
	"path/filepath"
)

func mkdir(dir string) {
	err := os.MkdirAll(dir, DIR_MODE)
	if err != nil {
		Dief(err.Error())
	}
	Logf("[success] mkdir -p %s", dir)
}

func wipe(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		Dief(err.Error())
	}
	Logf("[success] rm -rf %s", path)
}

func ditto(src string, dst string) {
	onFile := func(path string, f os.FileInfo, err error) error {
		if err != nil {
			Logf("ignoring error %s", err.Error())
			return nil
		}

		rel, err := filepath.Rel(src, path)
		must(err)

		dstpath := filepath.Join(dst, rel)
		mode := f.Mode()

		switch {
		case mode.IsDir():
			must(os.MkdirAll(dstpath, DIR_MODE))

		case mode.IsRegular():
			dittoReg(path, dstpath, f)

		case (mode&os.ModeSymlink > 0):
			dittoSymlink(path, dstpath, f)
		}

		if *appArgs.verbose {
			Logf(" - %s", rel)
		}
		return nil
	}

	rootinfo, err := os.Lstat(src)
	must(err)

	if rootinfo.IsDir() {
		filepath.Walk(src, onFile)
	} else {
		onFile(src, rootinfo, nil)
	}
	Logf("[success] rsync -a")
}

func dittoReg(srcpath string, dstpath string, f os.FileInfo) {
	must(os.RemoveAll(dstpath))

	writer, err := os.OpenFile(dstpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode()|MODE_MASK)
	must(err)
	defer writer.Close()

	reader, err := os.Open(srcpath)
	must(err)
	defer reader.Close()

	io.Copy(writer, reader)
}

func dittoSymlink(srcpath string, dstpath string, f os.FileInfo) {
	must(os.RemoveAll(dstpath))

	linkname, err := os.Readlink(srcpath)
	must(err)
	must(os.Symlink(linkname, dstpath))
}
