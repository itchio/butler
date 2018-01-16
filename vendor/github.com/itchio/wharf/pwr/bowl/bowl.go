package bowl

import (
	"fmt"
	"io"
)

type Bowl interface {
	// phase 1: patching
	GetWriter(index int64) (EntryWriter, error)
	Transpose(transposition Transposition) error

	// phase 2: committing
	Commit() error
}

type EntryWriter interface {
	Resume(checkpoint *Checkpoint) (int64, error)
	Save() (*Checkpoint, error)
	Tell() int64
	io.WriteCloser
}

var ErrUninitializedWriter = fmt.Errorf("tried to write to source before Resume() was called")

type Checkpoint struct {
	Offset int64
	Data   interface{}
}

type Transposition struct {
	TargetIndex int64
	SourceIndex int64
}
