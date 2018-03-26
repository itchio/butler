package wire

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

// WriteContext holds state of a wharf wire format writer
type WriteContext struct {
	writer io.Writer

	varintBuffer []byte
}

// NewWriteContext builds a new WriteContext that writes to a given writer
func NewWriteContext(writer io.Writer) *WriteContext {
	return &WriteContext{writer, make([]byte, 8)}
}

// Writer returns writer a WriteContext writes to
func (w *WriteContext) Writer() io.Writer {
	return w.writer
}

// Close closes the underlying writer if it implements io.Closer
func (w *WriteContext) Close() error {
	if c, ok := w.writer.(io.Closer); ok {
		err := c.Close()
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// WriteMagic writes a 32-bit magic integer to identify the file's type
func (w *WriteContext) WriteMagic(magic int32) error {
	return binary.Write(w.writer, Endianness, magic)
}

// WriteMessage serializes a protobuf message and writes it to the stream
func (w *WriteContext) WriteMessage(msg proto.Message) error {
	if DebugWire {
		fmt.Printf("<< %s %+v\n", reflect.TypeOf(msg).Elem().Name(), msg)
	}

	buf, err := proto.Marshal(msg)
	if err != nil {
		return errors.WithStack(err)
	}

	vibuflen := binary.PutUvarint(w.varintBuffer, uint64(len(buf)))
	if err != nil {
		return errors.WithStack(err)
	}
	_, err = w.writer.Write(w.varintBuffer[:vibuflen])
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = w.writer.Write(buf)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
