package splitfunc

import (
	"bufio"
	"io"
)

func New(blockSize int) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// still have more data than blockSize ? return a block-full
		if len(data) >= blockSize {
			return blockSize, data[:blockSize], nil
		}

		if atEOF {
			// at eof, but still have data: return all of it (must be <= blockSize)
			if len(data) > 0 {
				return len(data), data, nil
			}

			// at eof, no data left, signal EOF ourselves.
			return 0, nil, io.EOF
		}

		// wait for more data
		return 0, nil, nil
	}
}
