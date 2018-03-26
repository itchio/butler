package blockpool

import (
	"fmt"
	"io"

	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
)

// A BlockPool implements a pool that maps reads, seeks, and writes to blocks
type BlockPool struct {
	Container *tlc.Container

	Upstream   Source
	Downstream Sink
	Consumer   *state.Consumer

	reader *Reader
}

var _ wsync.Pool = (*BlockPool)(nil)
var _ wsync.WritablePool = (*BlockPool)(nil)

func (np *BlockPool) GetSize(fileIndex int64) int64 {
	return np.Container.Files[fileIndex].Size
}

func (np *BlockPool) GetReader(fileIndex int64) (io.Reader, error) {
	return np.GetReadSeeker(fileIndex)
}

func (np *BlockPool) GetReadSeeker(fileIndex int64) (io.ReadSeeker, error) {
	if np.Upstream == nil {
		return nil, errors.WithStack(fmt.Errorf("BlockPool: no upstream"))
	}

	if np.reader != nil {
		if np.reader.fileIndex == fileIndex {
			return np.reader, nil
		}

		err := np.reader.Close()
		if err != nil {
			return nil, err
		}
		np.reader = nil
	}

	fileSize := np.Container.Files[fileIndex].Size

	np.reader = &Reader{
		pool:      np,
		fileIndex: fileIndex,

		offset:    0,
		size:      fileSize,
		numBlocks: ComputeNumBlocks(fileSize),

		blockIndex: -1,
		blockBuf:   make([]byte, BigBlockSize),
	}
	return np.reader, nil
}

func (np *BlockPool) GetWriter(fileIndex int64) (io.WriteCloser, error) {
	if np.Downstream == nil {
		return nil, errors.WithStack(fmt.Errorf("BlockPool: no downstream"))
	}

	npw := &Writer{
		Pool:      np,
		FileIndex: fileIndex,

		offset:   0,
		size:     np.Container.Files[fileIndex].Size,
		blockBuf: make([]byte, BigBlockSize),
	}
	return npw, nil
}

func (np *BlockPool) Close() error {
	if np.reader != nil {
		err := np.reader.Close()
		if err != nil {
			return err
		}
		np.reader = nil
	}

	return nil
}
