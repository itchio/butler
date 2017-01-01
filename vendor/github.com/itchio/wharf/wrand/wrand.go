package wrand

import "math/rand"

// RandReader reads from rand.Source optimizedly
type RandReader struct {
	rand.Source
}

func (rr RandReader) Read(sink []byte) (int, error) {
	var tail, head int
	buf := make([]byte, 8)
	var r uint64
	for {
		head = min(tail+8, len(sink))
		if tail == head {
			return head, nil
		}

		r = (uint64)(rr.Int63())
		buf[0] = (byte)(r)
		buf[1] = (byte)(r >> 8)
		buf[2] = (byte)(r >> 16)
		buf[3] = (byte)(r >> 24)
		buf[4] = (byte)(r >> 32)
		buf[5] = (byte)(r >> 40)
		buf[6] = (byte)(r >> 48)
		buf[7] = (byte)(r >> 56)

		tail += copy(sink[tail:head], buf)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
