package bowl

import (
	"io"
	"os"
	"path/filepath"

	"github.com/itchio/savior"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
)

type freshBowl struct {
	TargetContainer *tlc.Container
	SourceContainer *tlc.Container
	TargetPath      string

	TargetPool wsync.Pool
	OutputPool *fspool.FsPool

	buf []byte
}

const freshBufferSize = 32 * 1024

var _ Bowl = (*freshBowl)(nil)

type FreshBowlParams struct {
	TargetContainer *tlc.Container
	SourceContainer *tlc.Container

	TargetPool   wsync.Pool
	OutputFolder string
}

// NewFreshBowl returns a bowl that applies all writes to
// a given (initially empty) directory.
func NewFreshBowl(params *FreshBowlParams) (Bowl, error) {
	// input validation

	if params.TargetContainer == nil {
		return nil, errors.New("freshbowl: TargetContainer must not be nil")
	}

	if params.TargetPool == nil {
		return nil, errors.New("freshbowl: TargetPool must not be nil")
	}

	if params.SourceContainer == nil {
		return nil, errors.New("freshbowl: SourceContainer must not be nil")
	}

	if params.OutputFolder == "" {
		return nil, errors.New("freshbowl: must specify OutputFolder")
	}

	outputPool := fspool.New(params.SourceContainer, params.OutputFolder)

	err := params.SourceContainer.Prepare(params.OutputFolder)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &freshBowl{
		TargetContainer: params.TargetContainer,
		SourceContainer: params.SourceContainer,
		TargetPool:      params.TargetPool,

		OutputPool: outputPool,
	}, nil
}

func (b *freshBowl) Save() (*BowlCheckpoint, error) {
	// we don't have anything to save
	c := &BowlCheckpoint{
		Data: nil,
	}
	return c, nil
}

func (b *freshBowl) Resume(c *BowlCheckpoint) error {
	// we don't have anything saved
	return nil
}

func (b *freshBowl) GetWriter(index int64) (EntryWriter, error) {
	return &freshEntryWriter{path: b.OutputPool.GetPath(index)}, nil
}

func (b *freshBowl) Transpose(t Transposition) (rErr error) {
	// alright y'all it's copy time

	r, err := b.TargetPool.GetReader(t.TargetIndex)
	if err != nil {
		rErr = errors.WithStack(err)
		return
	}

	w, err := b.OutputPool.GetWriter(t.SourceIndex)
	if err != nil {
		rErr = errors.WithStack(err)
		return
	}
	defer func() {
		cErr := w.Close()
		if cErr != nil && rErr == nil {
			rErr = errors.WithStack(cErr)
		}
	}()

	if len(b.buf) < freshBufferSize {
		b.buf = make([]byte, freshBufferSize)
	}

	_, err = io.CopyBuffer(w, r, b.buf)
	if err != nil {
		rErr = errors.WithStack(err)
		return
	}

	return
}

func (b *freshBowl) Commit() error {
	// it's all done buddy!
	return nil
}

// freshEntryWriter

type freshEntryWriter struct {
	f      *os.File
	path   string
	offset int64
}

var _ EntryWriter = (*freshEntryWriter)(nil)

func (few *freshEntryWriter) Tell() int64 {
	return few.offset
}

func (few *freshEntryWriter) Resume(c *WriterCheckpoint) (int64, error) {
	err := os.MkdirAll(filepath.Dir(few.path), 0755)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	f, err := os.OpenFile(few.path, os.O_CREATE|os.O_WRONLY, os.FileMode(0644))
	if err != nil {
		return 0, errors.WithStack(err)
	}

	if c != nil && c.Offset != 0 {
		_, err = f.Seek(c.Offset, io.SeekStart)
		if err != nil {
			return 0, errors.WithStack(err)
		}

		savior.Debugf("freshEntryWriter resuming at %d", c.Offset)
		few.offset = c.Offset
	}

	few.f = f
	return few.offset, nil
}

func (few *freshEntryWriter) Save() (*WriterCheckpoint, error) {
	err := few.f.Sync()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	savior.Debugf("freshEntryWriter saving at %d", few.offset)
	return &WriterCheckpoint{
		Offset: few.offset,
	}, nil
}

func (few *freshEntryWriter) Write(buf []byte) (int, error) {
	if few.f == nil {
		return 0, errors.WithStack(ErrUninitializedWriter)
	}

	n, err := few.f.Write(buf)
	few.offset += int64(n)
	return n, err
}

func (few *freshEntryWriter) Finalize() error {
	return nil
}

func (few *freshEntryWriter) Close() error {
	if few.f == nil {
		return nil
	}

	f := few.f
	few.f = nil

	err := f.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
