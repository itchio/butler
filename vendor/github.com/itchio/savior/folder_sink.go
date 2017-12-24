package savior

import (
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/state"
)

const (
	// ModeMask is or'd with files walked by butler
	ModeMask = 0666

	// LuckyMode is used when wiping in last-chance mode
	LuckyMode = 0777

	// DirMode is the default mode for directories created by butler
	DirMode = 0755
)

var onWindows = runtime.GOOS == "windows"

type FolderSink struct {
	Directory string
	Consumer  *state.Consumer

	writer *entryWriter
}

var _ Sink = (*FolderSink)(nil)

func (fs *FolderSink) destPath(entry *Entry) string {
	return filepath.Join(fs.Directory, filepath.FromSlash(entry.CanonicalPath))
}

func (fs *FolderSink) Mkdir(entry *Entry) error {
	dstpath := fs.destPath(entry)

	dirstat, err := os.Lstat(dstpath)
	if err != nil {
		// main case - dir doesn't exist yet
		err = os.MkdirAll(dstpath, DirMode)
		if err != nil {
			return errors.Wrap(err, 1)
		}
		return nil
	}

	if dirstat.IsDir() {
		// is already a dir, good!
	} else {
		// is a file or symlink for example, turn into a dir
		err = os.Remove(dstpath)
		if err != nil {
			return errors.Wrap(err, 1)
		}
		err = os.MkdirAll(dstpath, DirMode)
		if err != nil {
			return errors.Wrap(err, 1)
		}
	}

	return nil
}

func (fs *FolderSink) createFile(entry *Entry) (*os.File, error) {
	dstpath := fs.destPath(entry)

	dirname := filepath.Dir(dstpath)
	err := os.MkdirAll(dirname, LuckyMode)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	stats, err := os.Lstat(dstpath)
	if err == nil {
		if stats.Mode()&os.ModeSymlink > 0 {
			// if it used to be a symlink, remove it
			err = os.RemoveAll(dstpath)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}
		}
	}

	flag := os.O_CREATE | os.O_WRONLY
	f, err := os.OpenFile(dstpath, flag, entry.Mode|ModeMask)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if stats != nil && !onWindows {
		// if file already existed, chmod it, just in case
		err = f.Chmod(entry.Mode | ModeMask)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	return f, nil
}

func (fs *FolderSink) GetWriter(entry *Entry) (EntryWriter, error) {
	f, err := fs.createFile(entry)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if entry.WriteOffset > 0 {
		_, err = f.Seek(entry.WriteOffset, io.SeekStart)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	err = fs.closeAllWriters()
	if err != nil {
		fs.Consumer.Warnf("folder_sink could not close last writer: %s", err.Error())
	}

	ew := &entryWriter{
		fs:    fs,
		f:     f,
		entry: entry,
	}
	fs.writer = ew

	return ew, nil
}

func (fs *FolderSink) Preallocate(entry *Entry) error {
	f, err := fs.createFile(entry)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	defer f.Close()

	err = f.Truncate(entry.UncompressedSize)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func (fs *FolderSink) Symlink(entry *Entry, linkname string) error {
	if onWindows {
		// on windows, write symlinks as regular files
		w, err := fs.GetWriter(entry)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		defer w.Close()

		_, err = w.Write([]byte(linkname))
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}

	// actual symlink code
	dstpath := fs.destPath(entry)

	err := os.RemoveAll(dstpath)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	dirname := filepath.Dir(dstpath)
	err = os.MkdirAll(dirname, LuckyMode)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = os.Symlink(linkname, dstpath)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	return nil
}

func (fs *FolderSink) Nuke() error {
	err := fs.closeAllWriters()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	// TODO: retry logic, a-la butler
	return os.RemoveAll(fs.Directory)
}

func (fs *FolderSink) closeAllWriters() error {
	if fs.writer != nil {
		err := fs.writer.Close()
		fs.writer = nil
		return err
	}

	return nil
}

type entryWriter struct {
	fs    *FolderSink
	f     *os.File
	entry *Entry
}

var _ EntryWriter = (*entryWriter)(nil)

func (ew *entryWriter) Write(buf []byte) (int, error) {
	if ew.f == nil {
		return 0, os.ErrClosed
	}

	n, err := ew.f.Write(buf)
	ew.entry.WriteOffset += int64(n)
	return n, err
}

func (ew *entryWriter) Close() error {
	if ew.f == nil {
		// already closed
		return nil
	}

	err := ew.f.Close()
	ew.f = nil
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func (ew *entryWriter) Sync() error {
	if ew.f == nil {
		return os.ErrClosed
	}

	return ew.f.Sync()
}
