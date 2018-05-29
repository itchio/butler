package bowl

import (
	"fmt"
	"io"
)

type Bowl interface {
	// phase 1: patching
	Resume(checkpoint *BowlCheckpoint) error
	Save() (*BowlCheckpoint, error)
	GetWriter(index int64) (EntryWriter, error)
	Transpose(transposition Transposition) error

	// phase 2: committing
	Commit() error
}

type EntryWriter interface {
	Resume(checkpoint *WriterCheckpoint) (int64, error)
	Save() (*WriterCheckpoint, error)
	Tell() int64
	Finalize() error
	io.WriteCloser
}

var ErrUninitializedWriter = fmt.Errorf("tried to write to source before Resume() was called")

type BowlCheckpoint struct {
	Data interface{}
}

type WriterCheckpoint struct {
	Offset int64
	Data   interface{}
}

type Transposition struct {
	TargetIndex int64
	SourceIndex int64
}
