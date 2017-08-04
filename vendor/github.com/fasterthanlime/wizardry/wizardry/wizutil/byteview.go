package wizutil

const maxBufLen = 128 * 1024 // 128KB buffer

// ByteView allows treating an io.ReaderAt as a byte
// array.
type ByteView struct {
	Input    *SliceReader
	LookBack int64

	buf       []byte
	bufOffset int64
	bufLen    int64
}

// Get returns the byte at index i, or -1 if we
// failed to read
func (bv *ByteView) Get(i int64) int {
	if bv.buf == nil {
		bv.buf = make([]byte, maxBufLen)
	}

	// already got it in buf?
	posInBuffer := i - bv.bufOffset
	if posInBuffer >= 0 && posInBuffer < bv.bufLen {
		return int(bv.buf[posInBuffer])
	}

	newOffset := max(0, i-bv.LookBack)
	newEnd := min(newOffset+maxBufLen-1, bv.Input.Size()-1)
	newBufLen := (newEnd - newOffset) + 1

	bv.bufOffset = newOffset
	bv.bufLen = newBufLen

	// don't got it in buf! must read.
	_, err := bv.Input.ReadAt(bv.buf[:bv.bufLen], bv.bufOffset)
	if err != nil {
		// that's pretty bad
		return -1
	}

	posInBuffer = i - bv.bufOffset
	return int(bv.buf[posInBuffer])
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
