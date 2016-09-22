package blockpool

import (
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/itchio/wharf/tlc"
)

////////////////
// Filter
////////////////

// A BlockFilter is a whitelist of blocks one may filter by when fetching or storing
type BlockFilter map[int64]map[int64]bool

// Set adds a block to the whitelist
func (bf BlockFilter) Set(location BlockLocation) {
	if bf[location.FileIndex] == nil {
		bf[location.FileIndex] = make(map[int64]bool)
	}
	bf[location.FileIndex][location.BlockIndex] = true
}

// Has queries whether a block or not is in the whitelist
func (bf BlockFilter) Has(location BlockLocation) bool {
	row := bf[location.FileIndex]
	if row == nil {
		return false
	}
	return row[location.BlockIndex]
}

// Stats returns a human-readable string containing size information for this filter
func (bf BlockFilter) Stats(container *tlc.Container) string {
	totalBlocks := int64(0)
	totalSize := int64(0)

	usedBlocks := int64(0)
	usedSize := int64(0)

	for i, f := range container.Files {
		numBlocks := (f.Size + BigBlockSize - 1) / BigBlockSize
		for j := int64(0); j < numBlocks; j++ {
			totalBlocks++
			size := BigBlockSize
			alignedSize := (j + 1) * BigBlockSize
			if alignedSize > f.Size {
				size = f.Size % BigBlockSize
			}

			totalBlocks++
			totalSize += size

			if bf.Has(BlockLocation{FileIndex: int64(i), BlockIndex: j}) {
				usedBlocks++
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

// A FilteringSource only passes Fetch calls to the underling Source that pass through
// the given Filter, and returns a 0-filled
type FilteringSource struct {
	Source Source
	Filter BlockFilter

	zeroBuf   []byte
	container *tlc.Container
}

var _ Source = (*FilteringSource)(nil)

// Clone returns a copy of this filtering source. It also clones the underlying source.
func (fs *FilteringSource) Clone() Source {
	return &FilteringSource{
		Source: fs.Source.Clone(),
		Filter: fs.Filter,
	}
}

// Fetch returns the underlying source's result if the given location is in the
// filter, or a buffer filled with null bytes (of the correct size) otherwise
func (fs *FilteringSource) Fetch(location BlockLocation) ([]byte, error) {
	if fs.Filter.Has(location) {
		return fs.Source.Fetch(location)
	}

	// when filtered, return null bytes (from a single buffer)
	if fs.zeroBuf == nil {
		fs.zeroBuf = make([]byte, BigBlockSize)
	}

	blockLen := BigBlockSize
	alignedSize := (location.BlockIndex + 1) * BigBlockSize
	fileSize := fs.GetContainer().Files[location.FileIndex].Size
	if alignedSize > fileSize {
		blockLen = fileSize % BigBlockSize
	}
	return fs.zeroBuf[:blockLen], nil
}

// GetContainer returns the tlc container associated with the underlying source
func (fs *FilteringSource) GetContainer() *tlc.Container {
	return fs.Source.GetContainer()
}

////////////////
// Sink
////////////////

// A FilteringSink only relays Store calls to the underlying Sink if they pass
// through the given Filter, otherwise it just ignores them.
type FilteringSink struct {
	Sink   Sink
	Filter BlockFilter
}

var _ Sink = (*FilteringSink)(nil)

// Clone returns a copy of this filtering sink, that stores into a copy of the underlying sink
func (fs *FilteringSink) Clone() Sink {
	return &FilteringSink{
		Sink:   fs.Sink.Clone(),
		Filter: fs.Filter,
	}
}

// Store stores a block, if it passes the Filter, otherwise it does nothing.
func (fs *FilteringSink) Store(location BlockLocation, data []byte) error {
	if fs.Filter.Has(location) {
		return fs.Sink.Store(location, data)
	}

	// when filtered, just discard the block
	return nil
}

// GetContainer returns the tlc container associated with the underlying sink
func (fs *FilteringSink) GetContainer() *tlc.Container {
	return fs.Sink.GetContainer()
}
