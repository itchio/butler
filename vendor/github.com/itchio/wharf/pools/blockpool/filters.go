package blockpool

import (
	"fmt"

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
