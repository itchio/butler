package blockpool

import (
	"fmt"

	"github.com/itchio/wharf/tlc"
	"github.com/pkg/errors"
)

// A BlockAddressMap translates a location ({fileIndex, blockIndex}) to an address (":algo/:hash/:size").
// Unlike BlockHashMap, it is not thread-safe
type BlockAddressMap map[int64]map[int64]string

// Set stores a block's address in the map, given its location. Calling Set
// concurrently will result in undefined behavior (don't do it).
func (bam BlockAddressMap) Set(loc BlockLocation, path string) {
	if bam[loc.FileIndex] == nil {
		bam[loc.FileIndex] = make(map[int64]string)
	}
	bam[loc.FileIndex][loc.BlockIndex] = path
}

// Get stores a block's address in the map, given its location. May return empty string.
func (bam BlockAddressMap) Get(loc BlockLocation) string {
	if bam[loc.FileIndex] == nil {
		return ""
	}
	return bam[loc.FileIndex][loc.BlockIndex]
}

// TranslateFileIndices adapts a BlockAddressMap from one container to another,
// equivalent container. For example: currentContainer could have been read from
// a zip file, and the desiredContainer could have been read directly from the
// filesystem, resulting in different file ordering. Since files are referred
// to by their indices in most wharf operations, the block address map needs to
// be adjusted.
func (bam BlockAddressMap) TranslateFileIndices(currentContainer *tlc.Container, desiredContainer *tlc.Container) (BlockAddressMap, error) {
	newBam := make(BlockAddressMap)
	pathToIndex := make(map[string]int)

	if len(desiredContainer.Files) != len(currentContainer.Files) {
		return nil, errors.WithStack(fmt.Errorf("current container has %d files, desired has %d", len(currentContainer.Files), len(desiredContainer.Files)))
	}

	for i, f := range desiredContainer.Files {
		pathToIndex[f.Path] = i
	}

	for _, f := range currentContainer.Files {
		fileIndex := pathToIndex[f.Path]
		numBlocks := ComputeNumBlocks(f.Size)
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
