package wire

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/itchio/savior"
	"github.com/pkg/errors"
)

var (
	// ErrFormat is returned when we find a magic number that isn't the one we expected
	ErrFormat = fmt.Errorf("wrong magic (invalid input file)")
)

// ReadContext holds state of a wharf wire format reader
type ReadContext struct {
	source savior.Source

	countingReader *countingReader
	offset         int64

	protoBuffer *proto.Buffer

	saveState               saveState
	sourceCheckpoint        *savior.SourceCheckpoint
	messageReaderCheckpoint *MessageReaderCheckpoint
}

var _ MessageReader = (*ReadContext)(nil)

type saveState int

const (
	saveStateIdle saveState = iota
	saveStateWaitingForSource
	saveStateHasSourceCheckpoint
)

// NewReadContext builds a new ReadContext that reads from a given reader
func NewReadContext(source savior.Source) *ReadContext {
	r := &ReadContext{
		source: source,

		offset: 0,

		protoBuffer: proto.NewBuffer(make([]byte, 32*1024)),
	}
	r.countingReader = &countingReader{r}

	source.SetSourceSaveConsumer(&savior.CallbackSourceSaveConsumer{
		OnSave: func(checkpoint *savior.SourceCheckpoint) error {
			savior.Debugf("wire.ReadContext: Source gave us a checkpoint!")

			r.sourceCheckpoint = checkpoint
			r.saveState = saveStateHasSourceCheckpoint

			// we're going to make a full checkpoint on every next read

			return nil
		},
	})

	return r
}

func (r *ReadContext) GetSource() savior.Source {
	return r.source
}

func (r *ReadContext) Resume(checkpoint *MessageReaderCheckpoint) error {
	r.saveState = saveStateIdle
	r.sourceCheckpoint = nil
	r.messageReaderCheckpoint = nil

	savior.Debugf("ReadContext: asked to resume (checkpoint is nil? %v)", checkpoint == nil)
	if checkpoint != nil {
		sourceCheckpoint := checkpoint.SourceCheckpoint
		if sourceCheckpoint == nil {
			return errors.New("wire.ReadContext: missing sourceCheckpoint")
		}

		sourceOffset, err := r.source.Resume(sourceCheckpoint)
		if err != nil {
			return errors.WithStack(err)
		}

		delta := checkpoint.Offset - sourceOffset
		if delta < 0 {
			return errors.Errorf("wire.ReadContext: source (%d) resumed after our offset (%d), can't recover from that", sourceOffset, checkpoint.Offset)
		}
		if delta > 0 {
			savior.Debugf("wire.ReadContext: Discarding %d to align source (%d) with our offset (%d)", delta, sourceOffset, checkpoint.Offset)
			err = savior.DiscardByRead(r.source, delta)
			if err != nil {
				return errors.WithStack(err)
			}
			savior.Debugf("wire.ReadContext: Discarded %d successfully!", delta)
		}

		r.offset = checkpoint.Offset

		return nil
	}

	sourceOffset, err := r.source.Resume(nil)
	if err != nil {
		return errors.WithStack(err)
	}

	if sourceOffset != 0 {
		return errors.Errorf("ReadContext: expected source to resume at 0 but got %d", sourceOffset)
	}

	r.offset = 0

	return nil
}

func (r *ReadContext) WantSave() {
	if r.saveState == saveStateIdle {
		savior.Debugf("wire.ReadContext: Asked source for checkpoint")
		r.source.WantSave()
		r.saveState = saveStateWaitingForSource
	}
}

func (r *ReadContext) PopCheckpoint() *MessageReaderCheckpoint {
	if r.saveState == saveStateHasSourceCheckpoint {
		c := &MessageReaderCheckpoint{
			Offset:           r.offset,
			SourceCheckpoint: r.sourceCheckpoint,
		}
		r.saveState = saveStateIdle
		r.sourceCheckpoint = nil
		savior.Debugf("wire.ReadContext: Popping MessageReaderCheckpoint! offset = %d", c.Offset)
		return c
	}

	return nil
}

// ExpectMagic returns an error if the next 32-bit int is not the magic number specified
func (r *ReadContext) ExpectMagic(magic int32) error {
	var readMagic int32
	err := binary.Read(r.countingReader, Endianness, &readMagic)
	if err != nil {
		return errors.WithStack(err)
	}

	if magic != readMagic {
		return errors.WithStack(ErrFormat)
	}

	return nil
}

// ReadMessage deserializes a protobuf message from the underlying reader
func (r *ReadContext) ReadMessage(msg proto.Message) error {
	savior.Debugf("wire.ReadContext: Reading message at %d", r.offset)

	length, err := binary.ReadUvarint(r.countingReader)
	if err != nil {
		return errors.WithStack(err)
	}

	msgBuf := r.protoBuffer.Bytes()
	if cap(msgBuf) < int(length) {
		msgBuf = make([]byte, nextPowerOf2(int(length)))
	}

	_, err = io.ReadFull(r.countingReader, msgBuf[:length])
	if err != nil {
		return errors.WithStack(err)
	}
	r.protoBuffer.SetBuf(msgBuf[:length])

	msg.Reset()

	err = r.protoBuffer.Unmarshal(msg)
	if err != nil {
		return errors.WithStack(err)
	}

	if DebugWire {
		fmt.Printf(">> %s %+v\n", reflect.TypeOf(msg).Elem().Name(), msg)
	}

	return nil
}

func nextPowerOf2(v int) int {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}

// countingReader

type countingReader struct {
	r *ReadContext
}

var _ io.Reader = (*countingReader)(nil)
var _ io.ByteReader = (*countingReader)(nil)

func (cr *countingReader) Read(buf []byte) (int, error) {
	n, err := cr.r.source.Read(buf)
	cr.r.offset += int64(n)
	return n, err
}

func (cr *countingReader) ReadByte() (byte, error) {
	b, err := cr.r.source.ReadByte()
	if err == nil {
		cr.r.offset++
	}
	return b, err
}
