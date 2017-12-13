package szah

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/sevenzip-go/sz"
	"github.com/itchio/wharf/archiver"
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

type entryKind int

const (
	// those are reverse-engineered from printing out
	// the 'attrib' param of entries in .zip and .7z files
	entryKindDir     entryKind = 0x4
	entryKindFile    entryKind = 0x8
	entryKindSymlink entryKind = 0xa
)

type entryInfo struct {
	kind entryKind
	mode os.FileMode
}

func decodeEntryInfo(item *sz.Item) *entryInfo {
	var kind = entryKindFile
	var mode os.FileMode = 0644

	if attr, ok := item.GetUInt64Property(sz.PidPosixAttrib); ok {
		var kindmask uint64 = 0x0000f000
		var modemask uint64 = 0x000001ff
		kind = entryKind((attr & kindmask) >> (3 * 4))
		mode = os.FileMode(attr&modemask) & archiver.LuckyMode
	}

	if attr, ok := item.GetUInt64Property(sz.PidAttrib); ok {
		var kindmask uint64 = 0xf0000000
		var modemask uint64 = 0x01ff0000
		kind = entryKind((attr & kindmask) >> (7 * 4))
		mode = os.FileMode((attr&modemask)>>(4*4)) & archiver.LuckyMode
	}

	if isDir, _ := item.GetBoolProperty(sz.PidIsDir); isDir {
		kind = entryKindDir
	} else if _, ok := item.GetStringProperty(sz.PidSymLink); ok {
		kind = entryKindSymlink
	}

	switch kind {
	case entryKindDir: // that's ok
	case entryKindFile: // that's ok too
	case entryKindSymlink: // that's fine
	default:
		// uh oh, default to file
		kind = entryKindFile
	}

	if kind == entryKindFile {
		mode |= archiver.ModeMask
	}

	return &entryInfo{
		kind: kind,
		mode: mode,
	}
}
