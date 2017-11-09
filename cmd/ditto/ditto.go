package ditto

import (
	"io"
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/archiver"
)

var args = struct {
	src *string
	dst *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("ditto", "Create a mirror (incl. symlinks) of a directory into another dir (rsync -az)").Hidden()
	args.src = cmd.Arg("src", "Directory to mirror").Required().String()
	args.dst = cmd.Arg("dst", "Path where to create a mirror").Required().String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(*args.src, *args.dst))
}

// Does not preserve users, nor permission, except the executable bit
func Do(src string, dst string) error {
	comm.Debugf("rsync -a %s %s", src, dst)

	totalSize := int64(0)
	doneSize := int64(0)
	oldProgress := 0.0

	inc := func(_ string, f os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		totalSize += f.Size()
		return nil
	}

	onFile := func(path string, f os.FileInfo, err error) error {
		if err != nil {
			comm.Logf("ignoring error %s", err.Error())
			return nil
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		comm.Result(&mansion.FileMirroredResult{
			Type: "entry",
			Path: rel,
		})

		dstpath := filepath.Join(dst, rel)
		mode := f.Mode()

		switch {
		case mode.IsDir():
			err := dittoMkdir(dstpath)
			if err != nil {
				return errors.Wrap(err, 0)
			}

		case mode.IsRegular():
			err := dittoReg(path, dstpath, os.FileMode(f.Mode()&archiver.LuckyMode|archiver.ModeMask))
			if err != nil {
				return errors.Wrap(err, 0)
			}

		case (mode&os.ModeSymlink > 0):
			err := dittoSymlink(path, dstpath, f)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}

		comm.Debug(rel)

		doneSize += f.Size()

		progress := float64(doneSize) / float64(totalSize)
		if progress-oldProgress > 0.01 {
			oldProgress = progress
			comm.Progress(progress)
		}

		return nil
	}

	rootinfo, err := os.Lstat(src)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if rootinfo.IsDir() {
		totalSize = 0
		comm.Logf("Counting files in %s...", src)
		filepath.Walk(src, inc)

		comm.Logf("Mirroring...")
		err = filepath.Walk(src, onFile)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	} else {
		totalSize = rootinfo.Size()
		err = onFile(src, rootinfo, nil)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	comm.EndProgress()
	return nil
}

func dittoMkdir(dstpath string) error {
	comm.Debugf("mkdir %s", dstpath)
	err := archiver.Mkdir(dstpath)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	return nil
}

func dittoReg(srcpath string, dstpath string, mode os.FileMode) error {
	comm.Debugf("cp -f %s %s", srcpath, dstpath)
	err := os.RemoveAll(dstpath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	writer, err := os.OpenFile(dstpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer writer.Close()

	reader, err := os.Open(srcpath)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer reader.Close()

	_, err = io.Copy(writer, reader)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = os.Chmod(dstpath, mode)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func dittoSymlink(srcpath string, dstpath string, f os.FileInfo) error {
	err := os.RemoveAll(dstpath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	linkname, err := os.Readlink(srcpath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Debugf("ln -s %s %s", linkname, dstpath)
	err = os.Symlink(linkname, dstpath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
