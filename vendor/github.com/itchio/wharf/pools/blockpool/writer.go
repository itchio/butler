package blockpool

import (
	"fmt"
	"io"

	"github.com/go-errors/errors"
)

type BlockPoolWriter struct {
	Pool      *BlockPool
	FileIndex int64

	offset   int64
	size     int64
	blockBuf []byte

	closed bool
}

var _ io.WriteCloser = (*BlockPoolWriter)(nil)

func (npw *BlockPoolWriter) Write(buf []byte) (int, error) {
	if npw.closed {
		return 0, fmt.Errorf("write to closed BlockPoolWriter")
	}

	bufOffset := int64(0)
	bytesLeft := int64(len(buf))
	blockSize := npw.Pool.BlockSize

	for bytesLeft > 0 {
		blockIndex := npw.offset / blockSize
		blockEnd := (blockIndex + 1) * blockSize

		writeEnd := npw.offset + bytesLeft
		if writeEnd > blockEnd {
			writeEnd = blockEnd
		}

		bytesWritten := writeEnd - npw.offset
		blockBufOffset := npw.offset % blockSize
		copy(npw.blockBuf[blockBufOffset:], buf[bufOffset:bufOffset+bytesWritten])

		if writeEnd%blockSize == 0 {
			err := npw.Pool.Downstream.Store(BlockLocation{FileIndex: npw.FileIndex, BlockIndex: blockIndex}, npw.blockBuf)
			if err != nil {
				return 0, errors.Wrap(err, 1)
			}
		}

		bufOffset += bytesWritten
		npw.offset += bytesWritten
		bytesLeft -= bytesWritten
	}

	return len(buf), nil
}

func (npw *BlockPoolWriter) Close() error {
	if npw.closed {
		return nil
	}

	npw.closed = true

	blockSize := npw.Pool.BlockSize
	blockBufOffset := npw.offset % blockSize

	if blockBufOffset > 0 {
		blockIndex := npw.offset / blockSize
		err := npw.Pool.Downstream.Store(BlockLocation{FileIndex: npw.FileIndex, BlockIndex: blockIndex}, npw.blockBuf[:blockBufOffset])
		if err != nil {
			return errors.Wrap(err, 1)
		}
	}

	return nil
}
