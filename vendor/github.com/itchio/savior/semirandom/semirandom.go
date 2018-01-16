package semirandom

import (
	"bytes"
	"io"
	"math/rand"
)

// const DefaultSeed = 0xfaadbeef
const DefaultSeed = 0xfeefeefa

func Write(output io.Writer, length int64, seed int64) error {
	lw := &limitedWriter{
		w:   output,
		max: length,
	}
	rng := rand.New(rand.NewSource(seed))

	var oldSeqs [][]byte

	for lw.NeedsMore() {
		var seq []byte

		if rng.Intn(100) >= 80 && len(oldSeqs) > 0 {
			// re-use old seq
			seq = oldSeqs[rng.Intn(len(oldSeqs))]
		} else {
			seqLength := rng.Intn(48 * 1024)
			seq := make([]byte, seqLength)
			for j := 0; j < seqLength; j++ {
				seq[j] = byte(rng.Intn(255))
			}
			oldSeqs = append(oldSeqs, seq)
		}

		numRepetitions := rng.Intn(24)
		for j := 0; j < numRepetitions; j++ {
			_, err := lw.Write(seq)
			if err != nil {
				return err
			}
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
