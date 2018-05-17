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

func (pb *poolBowl) GetWriter(index int64) (EntryWriter, error) {
	w, err := pb.OutputPool.GetWriter(index)
	if err != nil {
		return nil, err
	}

	pew := &poolEntryWriter{w: w}
	return pew, nil
}

func (pb *poolBowl) Transpose(t Transposition) (rErr error) {
	// alright y'all it's copy time

	r, err := pb.TargetPool.GetReader(t.TargetIndex)
	if err != nil {
		rErr = errors.WithStack(err)
		return
	}

	w, err := pb.OutputPool.GetWriter(t.SourceIndex)
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

	if len(pb.buf) < poolBufferSize {
		pb.buf = make([]byte, poolBufferSize)
	}

	_, err = io.CopyBuffer(w, r, pb.buf)
	if err != nil {
		rErr = errors.WithStack(err)
		return
	}

	return
}

func (pb *poolBowl) Commit() error {
	return pb.OutputPool.Close()
}

// poolEntryWriter

type poolEntryWriter struct {
	w io.WriteCloser

	offset int64
}

var _ EntryWriter = (*poolEntryWriter)(nil)

func (pew *poolEntryWriter) Tell() int64 {
	return pew.offset
}

func (pew *poolEntryWriter) Resume(c *Checkpoint) (int64, error) {
	if c != nil {
		return 0, errors.Errorf("poolEntryWriter does not support checkpoints")
	}

	return 0, nil
}

func (pew *poolEntryWriter) Save() (*Checkpoint, error) {
	return nil, errors.Errorf("poolEntryWriter does not support checkpoints")
}

func (pew *poolEntryWriter) Write(buf []byte) (int, error) {
	n, err := pew.w.Write(buf)
	pew.offset += int64(n)
	return n, err
}

func (pew *poolEntryWriter) Close() error {
	return pew.w.Close()
}
