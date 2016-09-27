package blockpool

import (
	"io"
	"os"
)

// A Reader provides an io.ReadSeeker on top of a blockpool, knowing
// which block to fetch for which read requests.
type Reader struct {
	pool      *BlockPool
	fileIndex int64

	offset     int64
	size       int64
	numBlocks  int64
	blockIndex int64
	blockBuf   []byte
}

var _ io.ReadSeeker = (*Reader)(nil)

func (npr *Reader) Read(buf []byte) (int, error) {
	blockIndex := npr.offset / BigBlockSize
	if npr.blockIndex != blockIndex {
		if blockIndex >= npr.numBlocks {
			return 0, io.EOF
		}

		npr.blockIndex = blockIndex
		loc := BlockLocation{FileIndex: npr.fileIndex, BlockIndex: blockIndex}
		blockSize := ComputeBlockSize(npr.size, npr.blockIndex)

		// FIXME: should we check readBytes here? it would break filtering sources though.
		_, err := npr.pool.Upstream.Fetch(loc, npr.blockBuf[:blockSize])
		if err != nil {
			return 0, err
		}
	}

	newOffset := npr.offset + int64(len(buf))
	if newOffset > npr.size {
		newOffset = npr.size
	}

	blockEnd := (npr.blockIndex + 1) * BigBlockSize
	if newOffset > blockEnd {
		newOffset = blockEnd
	}

	readSize := int(newOffset - npr.offset)
	blockStart := npr.blockIndex * BigBlockSize
	blockOffset := npr.offset - blockStart
	copy(buf, npr.blockBuf[blockOffset:])
	npr.offset = newOffset

	if readSize == 0 {
		return 0, io.EOF
	}

	return readSize, nil
}

// Seek moves the read head as specified by (offset, whence), see io.Seeker's doc
func (npr *Reader) Seek(offset int64, whence int) (int64, error) {
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

// Close is actually a no-op
func (npr *Reader) Close() error {
	return nil
}
