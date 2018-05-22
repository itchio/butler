package blockpool

import (
	"fmt"

	"github.com/itchio/httpkit/progress"

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
		numBlocks := ComputeNumBlocks(f.Size)
		for j := int64(0); j < numBlocks; j++ {
			size := ComputeBlockSize(f.Size, j)

			totalBlocks++
			totalSize += size

			if bf.Has(BlockLocation{FileIndex: int64(i), BlockIndex: j}) {
				usedBlocks++
				usedSize += size
			}
		}
	}

	return fmt.Sprintf("%d / %d blocks, %s / %s (%.2f%%)", usedBlocks, totalBlocks,
		progress.FormatBytes(usedSize), progress.FormatBytes(totalSize),
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

	container *tlc.Container

	totalReqs   int64
	allowedReqs int64
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
func (fs *FilteringSource) Fetch(location BlockLocation, data []byte) (int, error) {
	fs.totalReqs++

	if fs.Filter.Has(location) {
		fs.allowedReqs++
		return fs.Source.Fetch(location, data)
	}

	// when filtered, don't touch output buffer
	return 0, nil
}

// GetContainer returns the tlc container associated with the underlying source
func (fs *FilteringSource) GetContainer() *tlc.Container {
	return fs.Source.GetContainer()
}

// Stats returns a human-readable string containing information on the filter rate of this source
func (fs *FilteringSource) Stats() string {
	return fmt.Sprintf("%d / %d fetches allowed (%.2f%% filter rate)",
		fs.allowedReqs, fs.totalReqs, (1.0-float64(fs.allowedReqs)/float64(fs.totalReqs))*100.0)
}

////////////////
// Sink
////////////////

// A FilteringSink only relays Store calls to the underlying Sink if they pass
// through the given Filter, otherwise it just ignores them.
type FilteringSink struct {
	Sink   Sink
	Filter BlockFilter

	totalReqs   int64
	allowedReqs int64
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
	fs.totalReqs++

	if fs.Filter.Has(location) {
		fs.allowedReqs++
		return fs.Sink.Store(location, data)
	}

	// when filtered, just discard the block
	return nil
}

// GetContainer returns the tlc container associated with the underlying sink
func (fs *FilteringSink) GetContainer() *tlc.Container {
	return fs.Sink.GetContainer()
}

// Stats returns a human-readable string containing information on the filter rate of this source
func (fs *FilteringSink) Stats() string {
	return fmt.Sprintf("%d / %d stores allowed (%.2f%% filter rate)",
		fs.allowedReqs, fs.totalReqs, (1.0-float64(fs.allowedReqs)/float64(fs.totalReqs))*100.0)
}
