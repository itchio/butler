package pwr

import (
	"testing"

	"github.com/alecthomas/assert"
)

func Test_BlockMath(t *testing.T) {
	// number of blocks
	assert.Equal(t, int64(0), ComputeNumBlocks(0))
	assert.Equal(t, int64(1), ComputeNumBlocks(1))
	assert.Equal(t, int64(1), ComputeNumBlocks(BlockSize-1))
	assert.Equal(t, int64(1), ComputeNumBlocks(BlockSize))
	assert.Equal(t, int64(2), ComputeNumBlocks(BlockSize+1))
	assert.Equal(t, int64(2), ComputeNumBlocks(BlockSize*2-1))
	assert.Equal(t, int64(3), ComputeNumBlocks(BlockSize*2+1))

	// block sizes
	assert.Equal(t, BlockSize-1, ComputeBlockSize(BlockSize-1, 0))

	assert.Equal(t, BlockSize, ComputeBlockSize(BlockSize, 0))

	assert.Equal(t, BlockSize, ComputeBlockSize(BlockSize+1, 0))
	assert.Equal(t, int64(1), ComputeBlockSize(BlockSize+1, 1))

	assert.Equal(t, BlockSize, ComputeBlockSize(BlockSize*2-1, 0))
	assert.Equal(t, BlockSize-1, ComputeBlockSize(BlockSize*2-1, 1))

	assert.Equal(t, BlockSize, ComputeBlockSize(BlockSize*2+1, 0))
	assert.Equal(t, BlockSize, ComputeBlockSize(BlockSize*2+1, 1))
	assert.Equal(t, int64(1), ComputeBlockSize(BlockSize*2+1, 2))
}
