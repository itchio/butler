package blockpool

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

// A Writer provides an io.Writer on top of a blockpool, storing data until
// it has enough to store a block downstream.
type Writer struct {
	Pool      *BlockPool
	FileIndex int64

	offset   int64
	size     int64
	blockBuf []byte

	closed bool
}

var _ io.WriteCloser = (*Writer)(nil)

// Write is an io.Writer-compliant implementation
func (npw *Writer) Write(buf []byte) (int, error) {
	if npw.closed {
		return 0, fmt.Errorf("write to closed Writer")
	}

	bufOffset := int64(0)
	bytesLeft := int64(len(buf))

	for bytesLeft > 0 {
		blockIndex := npw.offset / BigBlockSize
		blockEnd := (blockIndex + 1) * BigBlockSize

		writeEnd := npw.offset + bytesLeft
		if writeEnd > blockEnd {
			writeEnd = blockEnd
		}

		bytesWritten := writeEnd - npw.offset
		blockBufOffset := npw.offset % BigBlockSize
		copy(npw.blockBuf[blockBufOffset:], buf[bufOffset:bufOffset+bytesWritten])

		if writeEnd%BigBlockSize == 0 {
			err := npw.Pool.Downstream.Store(BlockLocation{FileIndex: npw.FileIndex, BlockIndex: blockIndex}, npw.blockBuf)
			if err != nil {
				return 0, errors.WithStack(err)
			}
		}

		bufOffset += bytesWritten
		npw.offset += bytesWritten
		bytesLeft -= bytesWritten
	}

	return len(buf), nil
}

// Close is essential, since it writes the last block in all cases except
// when it's an exact multiple of BigBlockSize
func (npw *Writer) Close() error {
	if npw.closed {
		return nil
	}

	npw.closed = true

	blockBufOffset := npw.offset % BigBlockSize

	if blockBufOffset > 0 {
		blockIndex := npw.offset / BigBlockSize
		err := npw.Pool.Downstream.Store(BlockLocation{FileIndex: npw.FileIndex, BlockIndex: blockIndex}, npw.blockBuf[:blockBufOffset])
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
