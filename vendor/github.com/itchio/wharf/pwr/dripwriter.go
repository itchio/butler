package pwr

import (
	"io"

	"github.com/go-errors/errors"
)

type ValidateFunc func(data []byte) error

// A DripWriter accepts Write()s of any size, buffers them, and relays only
// Writes of len(Buffer) to the underlying writer, calling Validate first
// if it's non-nil. The last Write might be < len(Buffer).
// The DripWriter must be closed, otherwise the last <= len(Buffer) bytes will be lost.
type DripWriter struct {
	Buffer   []byte
	Writer   io.Writer
	Validate ValidateFunc

	offset int
}

var _ io.WriteCloser = (*DripWriter)(nil)

func (dw *DripWriter) Write(data []byte) (int, error) {
	dataOffset := 0
	totalBytes := len(data)

	for dataOffset < totalBytes {
		writtenBytes := totalBytes - dataOffset
		if writtenBytes > len(dw.Buffer)-dw.offset {
			writtenBytes = len(dw.Buffer) - dw.offset
		}

		copy(dw.Buffer[dw.offset:], data[dataOffset:dataOffset+writtenBytes])
		dataOffset += writtenBytes
		dw.offset += writtenBytes

		if dw.offset == len(dw.Buffer) {
			buf := dw.Buffer

			if dw.Validate != nil {
				err := dw.Validate(buf)
				if err != nil {
					return 0, errors.Wrap(err, 1)
				}
			}

			_, err := dw.Writer.Write(buf)
			if err != nil {
				return 0, errors.Wrap(err, 1)
			}
			dw.offset = 0
		}
	}

	return totalBytes, nil
}

// Close acts as Flush + Close the underlying Writer, if it implements io.Closer
func (dw *DripWriter) Close() error {
	if dw.offset > 0 {
		buf := dw.Buffer[:dw.offset]

		if dw.Validate != nil {
			err := dw.Validate(buf)
			if err != nil {
				return errors.Wrap(err, 1)
			}
		}

		_, err := dw.Writer.Write(buf)
		if err != nil {
			return errors.Wrap(err, 1)
		}
		dw.offset = 0
	}

	if wc, ok := dw.Writer.(io.Closer); ok {
		err := wc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
