package szah

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/sevenzip-go/sz"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

var dontEnsureDeps = os.Getenv("BUTLER_NO_DEPS") == "1"
var ensuredDeps = false

type withArchiveCallback func(a *sz.Archive) error

func withArchive(consumer *state.Consumer, file eos.File, cb withArchiveCallback) error {
	err := ensureDeps(consumer)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	lib, err := sz.NewLib()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	defer lib.Free()

	stats, err := file.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	ext := strings.ToLower(filepath.Ext(stats.Name()))
	if ext != "" {
		ext = ext[1:] // strip "."
	}

	in, err := sz.NewInStream(file, ext, stats.Size())
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer in.Free()

	// try by extension first
	consumer.Debugf("Trying by extension '%s'", ext)
	a, err := lib.OpenArchive(in, false)
	if err != nil {
		// try by signature next
		_, err = in.Seek(0, io.SeekStart)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		consumer.Debugf("Trying by signature")
		a, err = lib.OpenArchive(in, true)
		if err != nil {
			// With the current libc7zip setup, 7-zip will refuse to
			// extract some self-extracting installers - for those,
			// we need to give it the `.cab` extension instead
			// Maybe the multivolume interface takes care of that?
			// Command-line `7z` has no issue with them.
			if ext == "exe" {
				consumer.Debugf("Trying by extension 'cab'")

				// if it was an .exe, try with a .cab extension
				in.Free()

				ext = "cab"

				in, err := sz.NewInStream(file, ext, stats.Size())
				if err != nil {
					return errors.Wrap(err, 0)
				}

				a, err = lib.OpenArchive(in, false) // by ext
				if err != nil {
					return errors.Wrap(err, 0)
				}
			} else {
				// well, we're out of options
				return errors.Wrap(err, 0)
			}
		}

	}

	return cb(a)
}
