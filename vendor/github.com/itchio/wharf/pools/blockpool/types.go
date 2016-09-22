package blockpool

import (
	"fmt"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/pwr"
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

// A BlockHashMap translates a location ({fileIndex, blockIndex}) to a hash
// it's usually read from a manifest file.
type BlockHashMap map[int64]map[int64][]byte

func (bhm BlockHashMap) Set(loc BlockLocation, data []byte) {
	if bhm[loc.FileIndex] == nil {
		bhm[loc.FileIndex] = make(map[int64][]byte)
	}
	bhm[loc.FileIndex][loc.BlockIndex] = data
}

func (bhm BlockHashMap) Get(loc BlockLocation) []byte {
	if bhm[loc.FileIndex] == nil {
		return nil
	}
	return bhm[loc.FileIndex][loc.BlockIndex]
}

// ToAddressMap translates block hashes to block addresses. It needs a container
// to compute blocks sizes (which is part of their address)
func (bhm BlockHashMap) ToAddressMap(container *tlc.Container, algorithm pwr.HashAlgorithm) (BlockAddressMap, error) {
	if algorithm != pwr.HashAlgorithm_SHAKE128_32 {
		return nil, errors.Wrap(fmt.Errorf("unsuported hash algorithm, want shake128-32, got %d", algorithm), 1)
	}

	bam := make(BlockAddressMap)
	for fileIndex, blocks := range bhm {
		f := container.Files[fileIndex]

		for blockIndex, hash := range blocks {
			size := BigBlockSize
			alignedSize := BigBlockSize * blockIndex
			if alignedSize > f.Size {
				size = f.Size % BigBlockSize
			}
			addr := fmt.Sprintf("shake128-32/%x/%d", hash, size)
			bam.Set(BlockLocation{FileIndex: fileIndex, BlockIndex: blockIndex}, addr)
		}
	}

	return bam, nil
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
