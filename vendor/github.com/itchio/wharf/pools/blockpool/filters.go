package blockpool

import (
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/wharf/tlc"
)

////////////////
// Filter
////////////////

// A BlockFilter is a whitelist of blocks one may filter by when fetching or storing
type BlockFilter map[int64]map[int64]bool

func (bf BlockFilter) Set(location BlockLocation) {
	if bf[location.FileIndex] == nil {
		bf[location.FileIndex] = make(map[int64]bool)
	}
	bf[location.FileIndex][location.BlockIndex] = true
}

func (bf BlockFilter) Has(location BlockLocation) bool {
	row := bf[location.FileIndex]
	if row == nil {
		return false
	}
	return row[location.BlockIndex]
}

func (bf BlockFilter) Stats(container *tlc.Container, blockSize int64) string {
	totalBlocks := int64(0)
	totalSize := int64(0)

	usedBlocks := int64(0)
	usedSize := int64(0)

	for i, f := range container.Files {
		numBlocks := f.Size / blockSize
		for j := int64(0); j < numBlocks; j++ {
			totalBlocks++
			size := blockSize
			alignedSize := (j + 1) * blockSize
			if alignedSize > f.Size {
				size = f.Size % blockSize
			}

			totalBlocks += 1
			totalSize += size

			if bf.Has(BlockLocation{FileIndex: int64(i), BlockIndex: j}) {
				usedBlocks += 1
				usedSize += size
			}
		}
	}

	return fmt.Sprintf("%d / %d blocks, %s / %s (%.2f%%)", usedBlocks, totalBlocks,
		humanize.IBytes(uint64(usedSize)), humanize.IBytes(uint64(totalSize)),
		float64(usedSize)/float64(totalSize)*100.0)
}

////////////////
// Source
////////////////

type FilteringSource struct {
	Source    Source
	Filter    BlockFilter
	BlockSize int64

	zeroBuf   []byte
	container *tlc.Container
}

var _ Source = (*FilteringSource)(nil)

func (fs *FilteringSource) Fetch(location BlockLocation) ([]byte, error) {
	if fs.Filter.Has(location) {
		return fs.Source.Fetch(location)
	}

	// when filtered, return null bytes (from a single buffer)
	if fs.zeroBuf == nil {
		if fs.BlockSize == 0 {
			return nil, errors.Wrap(fmt.Errorf("expected non-zero BlockSize"), 1)
		}
		fs.zeroBuf = make([]byte, fs.BlockSize)
	}

	blockLen := fs.BlockSize
	alignedSize := (location.BlockIndex + 1) * fs.BlockSize
	fileSize := fs.GetContainer().Files[location.FileIndex].Size
	if alignedSize > fileSize {
		blockLen = fileSize % fs.BlockSize
	}
	return fs.zeroBuf[:blockLen], nil
}

func (fs *FilteringSource) GetContainer() *tlc.Container {
	return fs.Source.GetContainer()
}

////////////////
// Sink
////////////////

type FilteringSink struct {
	Sink   Sink
	Filter BlockFilter
}

var _ Sink = (*FilteringSink)(nil)

func (fs *FilteringSink) Store(location BlockLocation, data []byte) error {
	if fs.Filter.Has(location) {
		return fs.Sink.Store(location, data)
	}

	// when filtered, just discard the block
	return nil
}

func (fs *FilteringSink) GetContainer() *tlc.Container {
	return fs.Sink.GetContainer()
}
