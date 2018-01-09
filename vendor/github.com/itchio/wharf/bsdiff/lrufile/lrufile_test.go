package lrufile_test

import (
	"bytes"
	"io"
	"math/rand"
	"testing"

	"github.com/itchio/wharf/bsdiff/lrufile"
	"github.com/stretchr/testify/assert"
)

func TestLruFile(t *testing.T) {
	dataSize := 16*1024 + 14
	rng := rand.New(rand.NewSource(0xfaaf))

	data := make([]byte, dataSize)
	_, err := rng.Read(data)
	must(t, err)

	r := bytes.NewReader(data)

	lf, err := lrufile.New(2721, 8)
	must(t, err)

	err = lf.Reset(r)
	must(t, err)

	outBuf := new(bytes.Buffer)

	stageBuf := make([]byte, 1731)

	_, err = io.CopyBuffer(outBuf, lf, stageBuf)
	must(t, err)

	assert.EqualValues(t, outBuf.Bytes(), data)

	t.Logf("Hits: %d Misses: %d", lf.Stats().Hits, lf.Stats().Misses)
}

func must(t *testing.T, err error) {
	if err != nil {
		assert.NoError(t, err)
		t.FailNow()
	}
}
