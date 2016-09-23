package blockpool

// ComputeNumBlocks returns the number of big blocks in a file, given its size
func ComputeNumBlocks(fileSize int64) int64 {
	return (fileSize + BigBlockSize - 1) / BigBlockSize
}

// ComputeBlockSize returns the size of a block in a file, given its location
func ComputeBlockSize(fileSize int64, blockIndex int64) int64 {
	if BigBlockSize*(blockIndex+1) > fileSize {
		return fileSize % BigBlockSize
	}
	return BigBlockSize
}
