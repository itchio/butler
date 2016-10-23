package blockpool

import (
	"testing"

	"github.com/alecthomas/assert"
)

func Test_BlockMath(t *testing.T) {
	// number of blocks
	assert.Equal(t, int64(0), ComputeNumBlocks(0))
	assert.Equal(t, int64(1), ComputeNumBlocks(1))
	assert.Equal(t, int64(1), ComputeNumBlocks(BigBlockSize-1))
	assert.Equal(t, int64(1), ComputeNumBlocks(BigBlockSize))
	assert.Equal(t, int64(2), ComputeNumBlocks(BigBlockSize+1))
	assert.Equal(t, int64(2), ComputeNumBlocks(BigBlockSize*2-1))
	assert.Equal(t, int64(3), ComputeNumBlocks(BigBlockSize*2+1))

	// block sizes
	assert.Equal(t, BigBlockSize-1, ComputeBlockSize(BigBlockSize-1, 0))

	assert.Equal(t, BigBlockSize, ComputeBlockSize(BigBlockSize, 0))

	assert.Equal(t, BigBlockSize, ComputeBlockSize(BigBlockSize+1, 0))
	assert.Equal(t, int64(1), ComputeBlockSize(BigBlockSize+1, 1))

	assert.Equal(t, BigBlockSize, ComputeBlockSize(BigBlockSize*2-1, 0))
	assert.Equal(t, BigBlockSize-1, ComputeBlockSize(BigBlockSize*2-1, 1))

	assert.Equal(t, BigBlockSize, ComputeBlockSize(BigBlockSize*2+1, 0))
	assert.Equal(t, BigBlockSize, ComputeBlockSize(BigBlockSize*2+1, 1))
	assert.Equal(t, int64(1), ComputeBlockSize(BigBlockSize*2+1, 2))
}
