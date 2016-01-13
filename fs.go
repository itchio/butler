package main

import (
	"io"
	"os"
	"path/filepath"
)

func mkdir(dir string) {
	if *appArgs.verbose {
		Logf("mkdir -p %s", dir)
	}

	err := os.MkdirAll(dir, DIR_MODE)
	if err != nil {
		Dief(err.Error())
	}
}

func wipe(path string) {
	if *appArgs.verbose {
		Logf("rm -rf %s", path)
	}

	err := os.RemoveAll(path)
	if err != nil {
		Dief(err.Error())
	}
}

func ditto(src string, dst string) {
	if *appArgs.verbose {
		Logf("rsync -a %s %s", src, dst)
	}

	totalSize := int64(0)
	doneSize := int64(0)
	oldPerc := 0.0

	inc := func(_ string, f os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		totalSize += f.Size()
		return nil
	}

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
			dittoMkdir(dstpath)

		case mode.IsRegular():
			dittoReg(path, dstpath, f)

		case (mode&os.ModeSymlink > 0):
			dittoSymlink(path, dstpath, f)
		}

		if *appArgs.verbose {
			Logf(" - %s", rel)
		}

		doneSize += f.Size()

		perc := float64(doneSize) / float64(totalSize) * 100.0
		if perc-oldPerc > 1.0 {
			oldPerc = perc
			Progress(perc)
		}

		return nil
	}

	rootinfo, err := os.Lstat(src)
	must(err)

	if rootinfo.IsDir() {
		totalSize = 0
		if !*appArgs.quiet {
			Logf("counting files in %s...", src)
		}
		filepath.Walk(src, inc)
		if !*appArgs.quiet {
			Logf("mirroring...")
		}
		filepath.Walk(src, onFile)
	} else {
		totalSize = rootinfo.Size()
		onFile(src, rootinfo, nil)
	}
}

func dittoMkdir(dstpath string) {
	if *appArgs.verbose {
		Logf("mkdir %s", dstpath)
	}

	dirstat, err := os.Lstat(dstpath)
	if err != nil {
		// main case - dir doesn't exist yet
		must(os.MkdirAll(dstpath, DIR_MODE))
		return
	}

	if !dirstat.IsDir() {
		// is a file or symlink for example, turn into a dir
		must(os.Remove(dstpath))
		must(os.MkdirAll(dstpath, DIR_MODE))
		return
	}

	// is already a dir, good!
}

func dittoReg(srcpath string, dstpath string, f os.FileInfo) {
	if *appArgs.verbose {
		Logf("cp -f %s %s", srcpath, dstpath)
	}
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

	if *appArgs.verbose {
		Logf("ln -s %s %s", linkname, dstpath)
	}
	must(os.Symlink(linkname, dstpath))
}
