package blockpool

import (
	"fmt"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/tlc"
)

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

func (bam BlockAddressMap) TranslateFileIndices(currentContainer *tlc.Container, desiredContainer *tlc.Container) (BlockAddressMap, error) {
	newBam := make(BlockAddressMap)
	pathToIndex := make(map[string]int)

	if len(desiredContainer.Files) != len(currentContainer.Files) {
		return nil, errors.Wrap(fmt.Errorf("current container has %d files, desired has %d", len(currentContainer.Files), len(desiredContainer.Files)), 1)
	}

	for i, f := range desiredContainer.Files {
		pathToIndex[f.Path] = i
	}

	for _, f := range currentContainer.Files {
		fileIndex := pathToIndex[f.Path]
		numBlocks := (f.Size + BigBlockSize - 1) / BigBlockSize
		for blockIndex := int64(0); blockIndex < numBlocks; blockIndex++ {
			loc := BlockLocation{FileIndex: int64(fileIndex), BlockIndex: blockIndex}
			addr := bam.Get(loc)
			if addr != "" {
				newBam.Set(loc, addr)
			}
		}
	}

	return newBam, nil
}

// A BlockPlace determines where a block lies in a given container
// the container's file are ordered, so FileIndex is reliable
type BlockLocation struct {
	FileIndex  int64
	BlockIndex int64
}
