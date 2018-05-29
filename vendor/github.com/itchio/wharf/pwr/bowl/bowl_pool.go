package bowl

import (
	"io"

	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
)

type poolBowl struct {
	TargetContainer *tlc.Container
	SourceContainer *tlc.Container
	TargetPath      string

	TargetPool wsync.Pool
	OutputPool wsync.WritablePool

	buf []byte
}

const poolBufferSize = 32 * 1024

var _ Bowl = (*poolBowl)(nil)

type PoolBowlParams struct {
	TargetContainer *tlc.Container
	SourceContainer *tlc.Container

	TargetPool wsync.Pool
	OutputPool wsync.WritablePool
}

// NewPoolBowl returns a bowl that applies all writes to
// a writable pool
func NewPoolBowl(params *PoolBowlParams) (Bowl, error) {
	// input validation

	if params.TargetContainer == nil {
		return nil, errors.New("poolBowl: TargetContainer must not be nil")
	}

	if params.TargetPool == nil {
		return nil, errors.New("poolBowl: TargetPool must not be nil")
	}

	if params.SourceContainer == nil {
		return nil, errors.New("poolBowl: SourceContainer must not be nil")
	}

	if params.OutputPool == nil {
		return nil, errors.New("poolBowl: must specify OutputFolder")
	}

	return &poolBowl{
		TargetContainer: params.TargetContainer,
		SourceContainer: params.SourceContainer,
		TargetPool:      params.TargetPool,
		OutputPool:      params.OutputPool,
	}, nil
}

func (b *poolBowl) Save() (*BowlCheckpoint, error) {
	return nil, errors.Errorf("poolBowl does not support checkpointing")
}

func (b *poolBowl) Resume(c *BowlCheckpoint) error {
	if c != nil {
		return errors.Errorf("poolBowl does not support checkpointing")
	}
	return nil
}

func (b *poolBowl) GetWriter(index int64) (EntryWriter, error) {
	w, err := b.OutputPool.GetWriter(index)
	if err != nil {
		return nil, err
	}

	pew := &poolEntryWriter{poolWriter: w}
	return pew, nil
}

func (b *poolBowl) Transpose(t Transposition) (rErr error) {
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

	if len(b.buf) < poolBufferSize {
		b.buf = make([]byte, poolBufferSize)
	}

	_, err = io.CopyBuffer(w, r, b.buf)
	if err != nil {
		rErr = errors.WithStack(err)
		return
	}

	return
}

func (b *poolBowl) Commit() error {
	return b.OutputPool.Close()
}

// poolEntryWriter

type poolEntryWriter struct {
	poolWriter io.WriteCloser

	offset int64
}

var _ EntryWriter = (*poolEntryWriter)(nil)

func (w *poolEntryWriter) Tell() int64 {
	return w.offset
}

func (w *poolEntryWriter) Resume(c *WriterCheckpoint) (int64, error) {
	if c != nil {
		return 0, errors.Errorf("poolEntryWriter does not support checkpoints")
	}

	return 0, nil
}

func (w *poolEntryWriter) Save() (*WriterCheckpoint, error) {
	return nil, errors.Errorf("poolEntryWriter does not support checkpoints")
}

func (w *poolEntryWriter) Write(buf []byte) (int, error) {
	n, err := w.poolWriter.Write(buf)
	w.offset += int64(n)
	return n, err
}

func (w *poolEntryWriter) Finalize() error {
	return nil
}

func (w *poolEntryWriter) Close() error {
	return w.poolWriter.Close()
}
