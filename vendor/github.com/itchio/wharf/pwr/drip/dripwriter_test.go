package drip

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/alecthomas/assert"
	"github.com/itchio/wharf/counter"
)

func Test_Writer(t *testing.T) {
	dropSize := 16

	numShort := 0
	shortSize := 0
	validate := func(buf []byte) error {
		switch {
		case len(buf) == dropSize:
			if numShort > 0 {
				return fmt.Errorf("got full after short")
			}
			return nil
		case len(buf) < dropSize:
			if numShort > 0 {
				return fmt.Errorf("got second short (%d)", len(buf))
			}
			numShort++
			shortSize = len(buf)
		default:
			return fmt.Errorf("drop too large (%d > %d)", len(buf), dropSize)
		}

		return nil
	}

	buf := make([]byte, dropSize)
	countingWriter := counter.NewWriter(nil)

	dw := &Writer{
		Buffer:   buf,
		Validate: validate,
		Writer:   countingWriter,
	}

	rbuf := make([]byte, 128)

	write := func(l int) {
		written, wErr := dw.Write(rbuf[0:l])
		assert.Equal(t, l, written)
		assert.NoError(t, wErr)
	}

	write(12)
	write(4)
	write(10)
	write(6)
	write(16)
	write(64)
	write(5)

	assert.NoError(t, dw.Close())
	assert.Equal(t, 5, shortSize)
	assert.Equal(t, int64(12+4+10+6+16+64+5), countingWriter.Count())
}

type tracingWriter struct {
	closeCall bool
	writeErr  error
	closeErr  error
}

var _ io.WriteCloser = (*tracingWriter)(nil)

func (tw *tracingWriter) Write(buf []byte) (int, error) {
	return len(buf), tw.writeErr
}

func (tw *tracingWriter) Close() error {
	tw.closeCall = true
	return tw.closeErr
}

func Test_WriterClose(t *testing.T) {
	dropSize := 16

	t.Logf("validation error on close")

	buf := make([]byte, dropSize)
	var validateError = errors.New("validation error")
	var underWriteError = errors.New("underlying write error")
	var underCloseError = errors.New("underlying close error")

	tw := &tracingWriter{}
	dw := &Writer{
		Buffer: buf,
		Validate: func(buf []byte) error {
			return validateError
		},
		Writer: tw,
	}

	_, wErr := dw.Write([]byte{1, 2, 3, 4})
	assert.NoError(t, wErr)

	cErr := dw.Close()
	assert.Error(t, cErr)
	assert.Equal(t, validateError, cErr)
	assert.True(t, tw.closeCall)

	t.Logf("underlying write error on close")

	tw = &tracingWriter{
		writeErr: underWriteError,
	}
	dw = &Writer{
		Buffer: buf,
		Validate: func(buf []byte) error {
			return nil
		},
		Writer: tw,
	}

	_, wErr = dw.Write([]byte{1, 2, 3, 4})
	assert.NoError(t, wErr)

	cErr = dw.Close()
	assert.Error(t, cErr)
	assert.Equal(t, underWriteError, cErr)
	assert.True(t, tw.closeCall)

	t.Logf("underlying close error on close")

	tw = &tracingWriter{
		closeErr: underCloseError,
	}
	dw = &Writer{
		Buffer: buf,
		Validate: func(buf []byte) error {
			return nil
		},
		Writer: tw,
	}

	_, wErr = dw.Write([]byte{1, 2, 3, 4})
	assert.NoError(t, wErr)

	cErr = dw.Close()
	assert.Error(t, cErr)
	assert.Equal(t, underCloseError, cErr)
	assert.True(t, tw.closeCall)
}
