package checker

import (
	"fmt"
	"io"
	"log"
	"math"

	"github.com/itchio/savior"
	"github.com/pkg/errors"
)

// TODO: check that everything has been extracted once properly

// Sink is a special kind of savior.Sink that records statistics
// about what's written to assist in unit testing.
type Sink struct {
	Items     map[string]*Item
	DoneItems map[string]*DoneItem
}

var _ savior.Sink = (*Sink)(nil)

// Item represents a savior.Entry + bytes pair
type Item struct {
	Entry *savior.Entry
	Data  []byte
}

// DoneItem represents the result of an item having been written to
type DoneItem struct {
	MinWrite int64
	MaxWrite int64
	Linkname string
}

// NewSink returns a new checker sink
func NewSink() *Sink {
	cs := &Sink{
		Items: make(map[string]*Item),
	}
	cs.Reset()
	return cs
}

func (cs *Sink) Reset() {
	cs.DoneItems = make(map[string]*DoneItem)
}

func (cs *Sink) Validate() error {
	numEntries := 0
	for _, i := range cs.Items {
		e := i.Entry
		if di, ok := cs.DoneItems[e.CanonicalPath]; ok {
			switch e.Kind {
			case savior.EntryKindFile:
				if di.MinWrite != 0 {
					return fmt.Errorf("checker.Sink: start of file missing (first write at byte %d): %s", di.MinWrite, e)
				}
				size := int64(len(i.Data))
				if di.MaxWrite != size {
					return fmt.Errorf("checker.Sink: end of file missing (last write ends at byte %d, should end at %d): %s", di.MaxWrite, size, e)
				}

			case savior.EntryKindSymlink:
				if di.Linkname != e.Linkname {
					return fmt.Errorf("checker.Sink: symlink points at '%s' instead of '%s': %s", di.Linkname, e.Linkname, e)
				}
			}
		} else {
			return fmt.Errorf("checker.Sink: entry neglected: %s", e)
		}
		numEntries++
	}
	log.Printf("checker.Sink: %d entries validated", numEntries)
	return nil
}

func (cs *Sink) Mkdir(entry *savior.Entry) error {
	return cs.withItem(entry, savior.EntryKindDir, func(item *Item, di *DoneItem) error {
		// that's about it
		return nil
	})
}

func (cs *Sink) Symlink(entry *savior.Entry, linkname string) error {
	return cs.withItem(entry, savior.EntryKindSymlink, func(item *Item, di *DoneItem) error {
		// that's about it
		if item.Entry.Linkname != linkname {
			err := fmt.Errorf("%s: expected dest '%s', got '%s'", entry.CanonicalPath, item.Entry.Linkname, linkname)
			return errors.WithStack(err)
		}

		di.Linkname = linkname

		return nil
	})
}

func (cs *Sink) GetWriter(entry *savior.Entry) (savior.EntryWriter, error) {
	var ew savior.EntryWriter

	err := cs.withItem(entry, savior.EntryKindFile, func(item *Item, di *DoneItem) error {
		checkingWriter := NewWriter(item.Data)
		checkingWriter.doneItem = di

		if entry.WriteOffset != 0 {
			_, err := checkingWriter.Seek(entry.WriteOffset, io.SeekStart)
			if err != nil {
				return errors.WithStack(err)
			}
		}

		ew = &entryWriter{
			entry: entry,
			w:     checkingWriter,
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return ew, nil
}

func (cs *Sink) Preallocate(entry *savior.Entry) error {
	return cs.withItem(entry, savior.EntryKindFile, func(item *Item, di *DoneItem) error {
		// nothing to do
		return nil
	})
}

func (cs *Sink) Nuke() error {
	cs.Reset()
	return nil
}

// ===============================

type withItemFunc func(item *Item, di *DoneItem) error

func (cs *Sink) withItem(entry *savior.Entry, actualKind savior.EntryKind, cb withItemFunc) error {
	item, ok := cs.Items[entry.CanonicalPath]
	if !ok {
		err := fmt.Errorf("%s: no such item", entry.CanonicalPath)
		return errors.WithStack(err)
	}

	expectedKind := item.Entry.Kind
	if item.Entry.Kind != actualKind {
		err := fmt.Errorf("%s: expected kind %v, got %v", entry.CanonicalPath, expectedKind, actualKind)
		return errors.WithStack(err)
	}

	di := cs.DoneItems[entry.CanonicalPath]
	if di == nil {
		di = &DoneItem{
			MinWrite: math.MaxInt64,
		}
		cs.DoneItems[entry.CanonicalPath] = di
	}

	return cb(item, di)
}

func (cs *Sink) Close() error {
	return nil
}

// ===============================

type entryWriter struct {
	w     io.Writer
	entry *savior.Entry
}

var _ savior.EntryWriter = (*entryWriter)(nil)

func (ew *entryWriter) Write(buf []byte) (int, error) {
	n, err := ew.w.Write(buf)
	ew.entry.WriteOffset += int64(n)
	return n, err
}

func (ew *entryWriter) Close() error {
	if closer, ok := ew.w.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (ew *entryWriter) Sync() error {
	return nil
}
