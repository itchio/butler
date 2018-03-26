package blockpool

import (
	"fmt"
	osync "sync"

	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/pkg/errors"
)

// A BlockHashMap maps a location ({fileIndex, blockIndex}) to a hash.
// It's usually read from a manifest file.
type BlockHashMap struct {
	mutex osync.Mutex
	data  map[int64]map[int64][]byte
}

// NewBlockHashMap creates a new empty BlockHashMap
func NewBlockHashMap() *BlockHashMap {
	return &BlockHashMap{
		data: make(map[int64]map[int64][]byte),
	}
}

// Set stores a block's hash in the map, given its location. Calling this
// concurrently is safe.
func (bhm *BlockHashMap) Set(loc BlockLocation, data []byte) {
	bhm.mutex.Lock()
	defer bhm.mutex.Unlock()

	if bhm.data[loc.FileIndex] == nil {
		bhm.data[loc.FileIndex] = make(map[int64][]byte)
	}
	bhm.data[loc.FileIndex][loc.BlockIndex] = data
}

// Get retrieves a block's hash in the map, given its location. May return nil
func (bhm *BlockHashMap) Get(loc BlockLocation) []byte {
	bhm.mutex.Lock()
	defer bhm.mutex.Unlock()

	if bhm.data[loc.FileIndex] == nil {
		return nil
	}
	return bhm.data[loc.FileIndex][loc.BlockIndex]
}

// ToAddressMap translates block hashes to block addresses. It needs a container
// to compute blocks sizes (which is part of their address)
func (bhm *BlockHashMap) ToAddressMap(container *tlc.Container, algorithm pwr.HashAlgorithm) (BlockAddressMap, error) {
	bhm.mutex.Lock()
	defer bhm.mutex.Unlock()

	if algorithm != pwr.HashAlgorithm_SHAKE128_32 {
		return nil, errors.WithStack(fmt.Errorf("unsuported hash algorithm, want shake128-32, got %d", algorithm))
	}

	bam := make(BlockAddressMap)
	for fileIndex, blocks := range bhm.data {
		f := container.Files[fileIndex]

		for blockIndex, hash := range blocks {
			size := ComputeBlockSize(f.Size, blockIndex)
			addr := fmt.Sprintf("shake128-32/%x/%d", hash, size)
			bam.Set(BlockLocation{FileIndex: fileIndex, BlockIndex: blockIndex}, addr)
		}
	}

	return bam, nil
}
