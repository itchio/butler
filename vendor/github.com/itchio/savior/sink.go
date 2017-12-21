package savior

import (
	"fmt"
	"io"
	"os"

	humanize "github.com/dustin/go-humanize"
)

type EntryKind int

const (
	// EntryKindDir is the kind for a directory
	EntryKindDir = 0
	// EntryKindSymlink is the kind for a symlink
	EntryKindSymlink = 1
	// EntryKindFile is the kind for a file
	EntryKindFile = 2
)

func (ek EntryKind) String() string {
	switch ek {
	case EntryKindDir:
		return "dir"
	case EntryKindSymlink:
		return "symlink"
	case EntryKindFile:
		return "file"
	default:
		return "<unknown entry kind>"
	}
}

// An Entry is a struct that should have *just the right fields*
// to be useful in an extractor checkpoint. They represent a file,
// directory, or symlink
type Entry struct {
	// CanonicalPath is a slash-separated path relative to the
	// root of the archive
	CanonicalPath string

	// Kind describes whether it's a regular file, a directory, or a symlink
	Kind EntryKind

	// Mode contains read/write/execute permissions, we're mostly interested in execute
	Mode os.FileMode

	// CompressedSize may be 0, if the extractor doesn't have the information
	CompressedSize int64

	// UncompressedSize may be 0, if the extractor doesn't have the information
	UncompressedSize int64

	// WriteOffset is useful if this entry struct is included in an extractor
	// checkpoint
	WriteOffset int64

	// Linkname describes the target of a symlink if the entry is a symlink
	// and the format we're extracting has symlinks in metadata rather than its contents
	Linkname string
}

func (entry *Entry) String() string {
	return fmt.Sprintf("%s (%s %s)", entry.CanonicalPath, humanize.IBytes(uint64(entry.UncompressedSize)), entry.Kind)
}

// An EntryWriter is an io.WriteCloser that you can Sync().
// This is important as saving a checkpoint (while in the middle of
// decompressing an archive) is only useful if we *know* that all
// the data we say we've decompressed is actually on disk (and not
// just stuck in a OS buffer somewhere).
type EntryWriter interface {
	io.WriteCloser

	// Sync should commit (to disk or otherwise) all the data written so far
	// to the entry.
	Sync() error
}

// A Sink is what extractors extract to. Typically, that would be
// a folder on a filesystem, but it could be anything else: repackaging
// as another archive type, uploading transparently as small blocks.
//
// Think of it as a very thin portion of `os.{Create,Mkdir,Symlink}` and
// friends that can be implemented completely independently of the filesystem
type Sink interface {
	// Mkdir creates a directory (and parents if needed)
	Mkdir(entry *Entry) error

	// Symlink creates a symlink
	Symlink(entry *Entry, linkname string) error

	// GetWriter returns a writer at entry.WriteOffset
	GetWriter(entry *Entry) (EntryWriter, error)

	// Preallocate space for a file based on the entry's UncompressedSize
	Preallocate(entry *Entry) error
}

// ===============================

func NopSync(w io.Writer) EntryWriter {
	return &nopSync{w: w}
}

type nopSync struct {
	w io.Writer
}

var _ EntryWriter = (*nopSync)(nil)

func (ns *nopSync) Write(buf []byte) (int, error) {
	return ns.w.Write(buf)
}

func (ns *nopSync) Close() error {
	if closer, ok := ns.w.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (ns *nopSync) Sync() error {
	return nil
}
