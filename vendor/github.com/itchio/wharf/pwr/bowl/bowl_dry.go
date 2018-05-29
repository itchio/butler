package bowl

import (
	"github.com/itchio/wharf/tlc"
	"github.com/pkg/errors"
)

type dryBowl struct {
	SourceContainer *tlc.Container
	TargetContainer *tlc.Container
}

var _ Bowl = (*dryBowl)(nil)

type DryBowlParams struct {
	SourceContainer *tlc.Container
	TargetContainer *tlc.Container
}

// NewDryBowl returns a bowl that throws away all writes
func NewDryBowl(params *DryBowlParams) (Bowl, error) {
	// input validation

	if params.TargetContainer == nil {
		return nil, errors.New("drybowl: TargetContainer must not be nil")
	}

	if params.SourceContainer == nil {
		return nil, errors.New("drybowl: SourceContainer must not be nil")
	}

	return &dryBowl{
		SourceContainer: params.SourceContainer,
		TargetContainer: params.TargetContainer,
	}, nil
}

func (b *dryBowl) Save() (*BowlCheckpoint, error) {
	// nothing to save
	c := &BowlCheckpoint{
		Data: nil,
	}
	return c, nil
}

func (b *dryBowl) Resume(c *BowlCheckpoint) error {
	// nothing saved
	return nil
}

func (b *dryBowl) GetWriter(index int64) (EntryWriter, error) {
	if index < 0 || index >= int64(len(b.SourceContainer.Files)) {
		return nil, errors.Errorf("drybowl: invalid source index %d", index)
	}

	// throw away the writes. alll the writes.
	return &nopEntryWriter{}, nil
}

func (b *dryBowl) Transpose(t Transposition) error {
	if t.SourceIndex < 0 || t.SourceIndex >= int64(len(b.SourceContainer.Files)) {
		return errors.Errorf("drybowl: invalid source index %d", t.SourceIndex)
	}
	if t.TargetIndex < 0 || t.TargetIndex >= int64(len(b.TargetContainer.Files)) {
		return errors.Errorf("drybowl: invalid target index %d", t.TargetIndex)
	}

	// muffin to do
	return nil
}

func (b *dryBowl) Commit() error {
	// literally nothing to do, we're just throwing stuff away!
	return nil
}

// nopEntryWriter

type nopEntryWriter struct {
	offset      int64
	initialized bool
}

var _ EntryWriter = (*nopEntryWriter)(nil)

func (new *nopEntryWriter) Tell() int64 {
	return new.offset
}

func (new *nopEntryWriter) Resume(c *WriterCheckpoint) (int64, error) {
	if c != nil {
		new.offset = c.Offset
	}

	new.initialized = true
	return new.offset, nil
}

func (new *nopEntryWriter) Save() (*WriterCheckpoint, error) {
	return &WriterCheckpoint{
		Offset: new.offset,
	}, nil
}

func (new *nopEntryWriter) Write(buf []byte) (int, error) {
	if !new.initialized {
		return 0, errors.WithStack(ErrUninitializedWriter)
	}

	new.offset += int64(len(buf))
	return len(buf), nil
}

func (new *nopEntryWriter) Finalize() error {
	return nil
}

func (new *nopEntryWriter) Close() error {
	return nil
}
