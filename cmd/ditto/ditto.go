package ditto

import (
	"io"
	"os"
	"path/filepath"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/archiver"
	"github.com/pkg/errors"
)

type Params struct {
	Src                 string
	Dst                 string
	PreservePermissions bool
}

var params Params

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("ditto", "Create a mirror (incl. symlinks) of a directory into another dir (rsync -az)").Hidden()
	cmd.Arg("src", "Directory to mirror").Required().StringVar(&params.Src)
	cmd.Arg("dst", "Path where to create a mirror").Required().StringVar(&params.Dst)
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(params))
}

// Does not preserve users, nor permission, except the executable bit
func Do(params Params) error {
	comm.Debugf("rsync -a %s %s", params.Src, params.Dst)

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

		rel, err := filepath.Rel(params.Src, path)
		if err != nil {
			return errors.WithStack(err)
		}

		comm.Result(&mansion.FileMirroredResult{
			Type: "entry",
			Path: rel,
		})

		dstpath := filepath.Join(params.Dst, rel)
		mode := f.Mode()

		switch {
		case mode.IsDir():
			err := dittoMkdir(dstpath)
			if err != nil {
				return errors.WithStack(err)
			}

		case mode.IsRegular():
			mode := f.Mode()
			if !params.PreservePermissions {
				mode = mode&archiver.LuckyMode | archiver.ModeMask
			}
			err := dittoReg(path, dstpath, os.FileMode(mode))
			if err != nil {
				return errors.WithStack(err)
			}

		case (mode&os.ModeSymlink > 0):
			err := dittoSymlink(path, dstpath, f)
			if err != nil {
				return errors.WithStack(err)
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

	rootinfo, err := os.Lstat(params.Src)
	if err != nil {
		return errors.WithStack(err)
	}

	if rootinfo.IsDir() {
		totalSize = 0
		comm.Logf("Counting files in %s...", params.Src)
		filepath.Walk(params.Src, inc)

		comm.Logf("Mirroring...")
		err = filepath.Walk(params.Src, onFile)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		totalSize = rootinfo.Size()
		err = onFile(params.Src, rootinfo, nil)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	comm.EndProgress()
	return nil
}

func dittoMkdir(dstpath string) error {
	comm.Debugf("mkdir %s", dstpath)
	err := archiver.Mkdir(dstpath)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func dittoReg(srcpath string, dstpath string, mode os.FileMode) error {
	comm.Debugf("cp -f %s %s", srcpath, dstpath)
	err := os.RemoveAll(dstpath)
	if err != nil {
		return errors.WithStack(err)
	}

	writer, err := os.OpenFile(dstpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return errors.WithStack(err)
	}
	defer writer.Close()

	reader, err := os.Open(srcpath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer reader.Close()

	_, err = io.Copy(writer, reader)
	if err != nil {
		return errors.WithStack(err)
	}

	err = os.Chmod(dstpath, mode)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func dittoSymlink(srcpath string, dstpath string, f os.FileInfo) error {
	err := os.RemoveAll(dstpath)
	if err != nil {
		return errors.WithStack(err)
	}

	linkname, err := os.Readlink(srcpath)
	if err != nil {
		return errors.WithStack(err)
	}

	comm.Debugf("ln -s %s %s", linkname, dstpath)
	err = os.Symlink(linkname, dstpath)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
