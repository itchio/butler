package blockpool

import "github.com/itchio/wharf/tlc"

// A Source allows obtaining the contents of a block
type Source interface {
	// Fetch retrieves a certain block, given its location. The returned buffer
	// must not be modified by the callee (ie. it must be a copy)
	Fetch(location BlockLocation) ([]byte, error)

	GetContainer() *tlc.Container
}

// A Sink lets one store a block
type Sink interface {
	// Store must fully read/copy/handle data before returning, as it might
	// be changed afterwards.
	Store(location BlockLocation, data []byte) error

	GetContainer() *tlc.Container
}

// A BlockAddressMap translates a location ({fileIndex, blockIndex}) to an address (":algo/:hash/:size")
// it's usually read from a manifest file
type BlockAddressMap map[int64]map[int64]string

func (bam BlockAddressMap) Set(loc BlockLocation, path string) {
	if bam[loc.FileIndex] == nil {
		bam[loc.FileIndex] = make(map[int64]string)
	}
	bam[loc.FileIndex][loc.BlockIndex] = path
}

func (bam BlockAddressMap) Get(loc BlockLocation) string {
	if bam[loc.FileIndex] == nil {
		return ""
	}
	return bam[loc.FileIndex][loc.BlockIndex]
}

// A BlockPlace determines where a block lies in a given container
// the container's file are ordered, so FileIndex is reliable
type BlockLocation struct {
	FileIndex  int64
	BlockIndex int64
}
