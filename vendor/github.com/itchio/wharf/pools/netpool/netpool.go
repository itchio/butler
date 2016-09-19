package netpool

import (
	"io"
	"os"

	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/sync"
	"github.com/itchio/wharf/tlc"
)

type BlockAddressMap map[int64]map[int64]string

func (bam BlockAddressMap) Set(fileIndex int64, blockIndex int64, path string) {
	if bam[fileIndex] == nil {
		bam[fileIndex] = make(map[int64]string)
	}
	bam[fileIndex][blockIndex] = path
}

type Source interface {
	Open(key string) (io.ReadCloser, error)
}

// A NetPool implements a pool that requests required blocks from network
type NetPool struct {
	Container      *tlc.Container
	BlockSize      int64
	BlockAddresses BlockAddressMap

	Upstream Source
	Consumer *pwr.StateConsumer

	reader *NetPoolReader
}

var _ sync.Pool = (*NetPool)(nil)

func (np *NetPool) GetReader(fileIndex int64) (io.Reader, error) {
	return np.GetReadSeeker(fileIndex)
}

func (np *NetPool) GetReadSeeker(fileIndex int64) (io.ReadSeeker, error) {
	if np.reader != nil {
		if np.reader.FileIndex == fileIndex {
			return np.reader, nil
		}

		err := np.reader.Close()
		if err != nil {
			return nil, err
		}
		np.reader = nil
	}

	np.reader = &NetPoolReader{
		Pool:      np,
		FileIndex: fileIndex,

		offset: 0,
		size:   np.Container.Files[fileIndex].Size,

		blockIndex: -1,
		blockBuf:   make([]byte, np.BlockSize),
	}
	return np.reader, nil
}

func (np *NetPool) Close() error {
	if np.reader != nil {
		err := np.reader.Close()
		if err != nil {
			return err
		}
		np.reader = nil
	}

	return nil
}

type NetPoolReader struct {
	Pool      *NetPool
	FileIndex int64

	offset     int64
	size       int64
	blockIndex int64
	blockBuf   []byte
}

var _ io.ReadSeeker = (*NetPoolReader)(nil)

func (npr *NetPoolReader) Read(buf []byte) (int, error) {
	blockIndex := npr.offset / npr.Pool.BlockSize
	if npr.blockIndex != blockIndex {
		npr.blockIndex = blockIndex
		address := npr.Pool.BlockAddresses[npr.FileIndex][npr.blockIndex]
		if address != "" {
			r, err := npr.Pool.Upstream.Open(address)
			if err != nil {
				return 0, err
			}
			io.ReadFull(r, npr.blockBuf)
		}
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

func (npr *NetPoolReader) Seek(offset int64, whence int) (int64, error) {
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

func (npr *NetPoolReader) Close() error {
	return nil
}
