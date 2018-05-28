package overlay

import (
	"bufio"
	"encoding/gob"
	"io"

	"github.com/itchio/wharf/counter"

	"github.com/pkg/errors"
)

const overlayBufSize = 128 * 1024     // 128KiB
const overlaySameThreshold = 8 * 1024 // 8KiB

type OverlayOp struct {
	Skip  int64
	Fresh int64
	Eof   bool
}

type overlayWriter struct {
	cw          *counter.Writer
	writeOffset int64

	r          io.Reader
	readOffset int64

	bw   *bufio.Writer
	rbuf []byte

	encoder *gob.Encoder
}

type overlayProcessor struct {
	ow *overlayWriter
}

type OverlayWriter interface {
	io.WriteCloser
	Flush() error

	ReadOffset() int64
	WriteOffset() int64
}

// NewOverlayWriter returns a writer that reads from `r` and only
// encodes changed data to `w`.
// Closing it will not close the underlying writer!
func NewOverlayWriter(r io.Reader, readOffset int64, w io.Writer, writeOffset int64) OverlayWriter {
	rbuf := make([]byte, overlayBufSize)

	ow := &overlayWriter{
		r:           r,
		readOffset:  readOffset,
		writeOffset: writeOffset,
		rbuf:        rbuf,
	}

	cw := counter.NewWriterCallback(ow.recordWrite, w)
	ow.cw = cw
	ow.encoder = gob.NewEncoder(cw)

	ow.bw = bufio.NewWriterSize(&overlayProcessor{ow}, overlayBufSize)

	return ow
}

func (ow *overlayWriter) recordWrite(count int64) {
	ow.writeOffset += count
}

func (ow *overlayWriter) Write(buf []byte) (int, error) {
	return ow.bw.Write(buf)
}

func (ow *overlayWriter) Flush() error {
	return ow.bw.Flush()
}

func (ow *overlayWriter) ReadOffset() int64 {
	return ow.readOffset
}

func (ow *overlayWriter) WriteOffset() int64 {
	return ow.writeOffset
}

func (op *overlayProcessor) Write(buf []byte) (int, error) {
	written := 0

	for written < len(buf) {
		blockWritten, err := op.write(buf)
		buf = buf[blockWritten:]
		written += blockWritten

		if err != nil {
			return written, errors.WithStack(err)
		}
	}

	return written, nil
}

func (op *overlayProcessor) write(buf []byte) (int, error) {
	ow := op.ow

	if len(buf) > overlayBufSize {
		buf = buf[:overlayBufSize]
	}
	rbuf := ow.rbuf

	// time to compare!
	rbuflen, err := ow.r.Read(rbuf[:len(buf)])
	if err != nil {
		if errors.Cause(err) == io.EOF {
			// EOF is fine, new file might be larger
		} else {
			return 0, errors.WithStack(err)
		}
	}

	{
		// find data we can skip
		var lastOp int
		var same int

		commit := func(i int) error {
			freshLen := i - same - lastOp
			if freshLen > 0 {
				err = ow.fresh(buf[lastOp : i-same])
				if err != nil {
					return errors.WithStack(err)
				}
				lastOp = i - same
			}

			lastOp = i
			err = ow.skip(int64(same))
			if err != nil {
				return errors.WithStack(err)
			}

			return nil
		}

		for i := 0; i < rbuflen; i++ {
			if rbuf[i] == buf[i] {
				// count the number of similar bytes as we go
				same++
			} else {
				if same > overlaySameThreshold {
					err := commit(i)
					if err != nil {
						return 0, errors.WithStack(err)
					}
				}
				same = 0
			}
		}

		i := rbuflen

		// did we finish on a same streak?
		if same > overlaySameThreshold {
			err := commit(i)
			if err != nil {
				return 0, errors.WithStack(err)
			}
		}

		// anything fresh left to write?
		if lastOp < i {
			err := ow.fresh(buf[lastOp:rbuflen])
			if err != nil {
				return 0, errors.WithStack(err)
			}
		}
	}

	// finally, if we have any trailing data, it's fresh
	if rbuflen < len(buf) {
		err = ow.fresh(buf[rbuflen:])
		if err != nil {
			return 0, errors.WithStack(err)
		}
	}

	return len(buf), nil
}

func (ow *overlayWriter) fresh(data []byte) error {
	op := &OverlayOp{
		Fresh: int64(len(data)),
	}

	err := ow.encoder.Encode(op)
	if err != nil {
		return errors.WithStack(err)
	}

	written, err := ow.cw.Write(data)
	if err != nil {
		return errors.WithStack(err)
	}

	if written < len(data) {
		return errors.Errorf("expected to write %d bytes, wrote %d", len(data), written)
	}

	ow.readOffset += int64(len(data))

	return nil
}

func (ow *overlayWriter) skip(skip int64) error {
	op := &OverlayOp{
		Skip: skip,
	}

	err := ow.encoder.Encode(op)
	if err != nil {
		return errors.WithStack(err)
	}

	ow.readOffset += skip

	return nil
}

func (ow *overlayWriter) Close() error {
	err := ow.Flush()
	if err != nil {
		return errors.WithStack(err)
	}

	op := &OverlayOp{
		Eof: true,
	}

	err = ow.encoder.Encode(op)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
