package blockpool

import (
	"io"
	"os"
)

type BlockPoolReader struct {
	Pool      *BlockPool
	FileIndex int64

	offset     int64
	size       int64
	blockIndex int64
	blockBuf   []byte
}

var _ io.ReadSeeker = (*BlockPoolReader)(nil)

func (npr *BlockPoolReader) Read(buf []byte) (int, error) {
	blockIndex := npr.offset / npr.Pool.BlockSize
	if npr.blockIndex != blockIndex {
		npr.blockIndex = blockIndex
		blockBuf, err := npr.Pool.Upstream.Fetch(BlockLocation{npr.FileIndex, blockIndex})
		if err != nil {
			return 0, err
		}
		npr.blockBuf = blockBuf
	}

	newOffset := npr.offset + int64(len(buf))
	if newOffset > npr.size {
		newOffset = npr.size
	}

	blockEnd := (npr.blockIndex + 1) * npr.Pool.BlockSize
	if newOffset > blockEnd {
		newOffset = blockEnd
	}

	readSize := int(newOffset - npr.offset)
	blockStart := npr.blockIndex * npr.Pool.BlockSize
	blockOffset := npr.offset - blockStart
	copy(buf, npr.blockBuf[blockOffset:])
	npr.offset = newOffset

	if readSize == 0 {
		return 0, io.EOF
	} else {
		return readSize, nil
	}
}

func (npr *BlockPoolReader) Seek(offset int64, whence int) (int64, error) {
	npr.Pool.Consumer.Debugf("seek(%d, %d)", offset, whence)
	switch whence {
	case os.SEEK_END:
		npr.offset = npr.size + offset
	case os.SEEK_CUR:
		npr.offset += offset
	case os.SEEK_SET:
		npr.offset = offset
	}
	return npr.offset, nil
}

func (npr *BlockPoolReader) Close() error {
	return nil
}
