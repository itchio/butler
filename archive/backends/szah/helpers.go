package szah

import (
	"io"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/sevenzip-go/sz"
	"github.com/itchio/wharf/eos"
)

type withArchiveCallback func(a *sz.Archive) error

func withArchive(path string, cb withArchiveCallback) error {
	lib, err := sz.NewLib()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	defer lib.Free()

	f, err := eos.Open(path)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	ext := filepath.Ext(path)
	if ext != "" {
		ext = ext[1:] // strip "."
	}

	in, err := sz.NewInStream(f, ext, info.Size())
	if err != nil {
		return errors.Wrap(err, 0)
	}

	// try by extension first
	a, err := lib.OpenArchive(in, false)
	if err != nil {
		// try by signature next
		_, err = in.Seek(0, io.SeekStart)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		a, err = lib.OpenArchive(in, true)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return cb(a)
}
