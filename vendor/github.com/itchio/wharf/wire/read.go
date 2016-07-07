package wire

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"

	"github.com/go-errors/errors"
	"github.com/golang/protobuf/proto"
)

var (
	// ErrFormat is returned when we find a magic number that isn't the one we expected
	ErrFormat = errors.New("wrong magic (invalid input file)")
)

// ReadContext holds state of a wharf wire format reader
type ReadContext struct {
	reader io.Reader

	byteBuffer []byte
	msgBuf     []byte
}

// NewReadContext builds a new ReadContext that reads from a given reader
func NewReadContext(reader io.Reader) *ReadContext {
	return &ReadContext{reader, make([]byte, 1), make([]byte, 32)}
}

// ReadByte reads a single byte from the underlying reader
func (r *ReadContext) ReadByte() (byte, error) {
	_, err := r.reader.Read(r.byteBuffer)
	if err != nil {
		return 0, err
	}

	return r.byteBuffer[0], nil
}

// Reader returns the underlying reader
func (r *ReadContext) Reader() io.Reader {
	return r.reader
}

// ExpectMagic returns an error if the next 32-bit int is not the magic number specified
func (r *ReadContext) ExpectMagic(magic int32) error {
	var readMagic int32
	err := binary.Read(r.reader, Endianness, &readMagic)
	if err != nil {
		return err
	}

	if magic != readMagic {
		return ErrFormat
	}

	return nil
}

// ReadMessage deserializes a protobuf message from the underlying reader
func (r *ReadContext) ReadMessage(msg proto.Message) error {
	length, err := binary.ReadUvarint(r)
	if err != nil {
		return err
	}

	if cap(r.msgBuf) < int(length) {
		r.msgBuf = make([]byte, length)
	}

	_, err = io.ReadFull(r.reader, r.msgBuf[:length])
	if err != nil {
		return err
	}

	err = proto.Unmarshal(r.msgBuf[:length], msg)
	if err != nil {
		return err
	}

	if DebugWire {
		fmt.Printf(">> %s %+v\n", reflect.TypeOf(msg).Elem().Name(), msg)
	}

	return nil
}
