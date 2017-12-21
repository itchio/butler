package checker

import (
	"fmt"
	"io"

	"github.com/go-errors/errors"
	"github.com/itchio/savior"
)

type Sink struct {
	Items map[string]*Item
}

var _ savior.Sink = (*Sink)(nil)

type Item struct {
	Entry    *savior.Entry
	Data     []byte
	Linkname string
}

func NewSink() *Sink {
	return &Sink{
		Items: make(map[string]*Item),
	}
}

func (cs *Sink) Mkdir(entry *savior.Entry) error {
	return cs.withItem(entry, savior.EntryKindDir, func(item *Item) error {
		// that's about it
		return nil
	})
}

func (cs *Sink) Symlink(entry *savior.Entry, linkname string) error {
	return cs.withItem(entry, savior.EntryKindSymlink, func(item *Item) error {
		// that's about it
		if item.Linkname != linkname {
			err := fmt.Errorf("%s: expected dest '%s', got '%s'", entry.CanonicalPath, item.Linkname, linkname)
			return errors.Wrap(err, 0)
		}

		return nil
	})
}

func (cs *Sink) GetWriter(entry *savior.Entry) (savior.EntryWriter, error) {
	var ew savior.EntryWriter

	err := cs.withItem(entry, savior.EntryKindFile, func(item *Item) error {
		c := NewWriter(item.Data)

		if entry.WriteOffset != 0 {
			_, err := c.Seek(entry.WriteOffset, io.SeekStart)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}

		ew = savior.NopSync(c)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return ew, nil
}

func (cs *Sink) Preallocate(entry *savior.Entry) error {
	return cs.withItem(entry, savior.EntryKindFile, func(item *Item) error {
		// nothing to do
		return nil
	})
}

// ===============================

type withItemFunc func(item *Item) error

func (cs *Sink) withItem(entry *savior.Entry, actualKind savior.EntryKind, cb withItemFunc) error {
	item, ok := cs.Items[entry.CanonicalPath]
	if !ok {
		err := fmt.Errorf("%s: no such item", entry.CanonicalPath)
		return errors.Wrap(err, 0)
	}

	expectedKind := item.Entry.Kind
	if item.Entry.Kind != actualKind {
		err := fmt.Errorf("%s: expected kind %v, got %v", entry.CanonicalPath, expectedKind, actualKind)
		return errors.Wrap(err, 0)
	}

	return cb(item)
}
