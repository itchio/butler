package wire

import (
	"encoding/gob"

	"github.com/golang/protobuf/proto"
	"github.com/itchio/savior"
)

type MessageReader interface {
	// Resume returns an error it was unable
	// to resume from the given checkpoint, and
	// returns nil otherwise.
	Resume(checkpoint *MessageReaderCheckpoint) error

	// ExpectMagic ensures that a given magic number is
	// contained in the wire format
	ExpectMagic(magic int32) error

	// ReadMessage reads a protobuf message
	ReadMessage(msg proto.Message) error

	// Signal that we want to save as soon as possible
	// This may be called any number of times
	WantSave()

	// PopCheckpoint returns the last made checkpoint
	// if ready, or nil if there are none ready
	PopCheckpoint() *MessageReaderCheckpoint
}

type MessageReaderCheckpoint struct {
	Offset           int64
	SourceCheckpoint *savior.SourceCheckpoint
}

func init() {
	gob.Register(&MessageReaderCheckpoint{})
}
