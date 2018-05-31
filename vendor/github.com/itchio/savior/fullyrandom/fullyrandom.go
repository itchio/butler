package fullyrandom

import (
	"bytes"
	"io"
	"math/rand"
)

const DefaultSeed = 0xfaadbeef

func Write(output io.Writer, length int64, seed int64) error {
	lw := &limitedWriter{
		w:   output,
		max: length,
	}
	rng := rand.New(rand.NewSource(seed))

	bufsize := 1024
	buf := make([]byte, bufsize)

	for lw.NeedsMore() {
		for i := 0; i < bufsize; i++ {
			buf[i] = byte(rng.Intn(255))
		}
		_, err := lw.Write(buf)
		if err != nil {
			return err
		}
	}

	return nil
}

func Bytes(length int64) []byte {
	buf := new(bytes.Buffer)
	Write(buf, length, DefaultSeed)
	return buf.Bytes()
}

type limitedWriter struct {
	w io.Writer

	c   int64
	max int64
}

var _ io.Writer = (*limitedWriter)(nil)

func (lw *limitedWriter) Write(buf []byte) (int, error) {
	if lw.c >= lw.max {
		// just drop it!
		return len(buf), nil
	}

	toWrite := len(buf)
	nextEnd := lw.c + int64(toWrite)
	if nextEnd > lw.max {
		toWrite = int(lw.max - lw.c)
	}

	n, err := lw.w.Write(buf[:toWrite])
	lw.c += int64(n)
	return n, err
}

func (lw *limitedWriter) NeedsMore() bool {
	return lw.c < lw.max
}
