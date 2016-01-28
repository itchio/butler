package main

import (
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/itchio/wharf/crc32c"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/stretchr/testify/assert"
)

func contents(info *tlc.RepoInfo, path string) []byte {
	r := info.NewReader(path)
	b, err := ioutil.ReadAll(r)
	must(err)
	return b
}

func dirHash(info *tlc.RepoInfo, path string) []byte {
	r := info.NewReader(path)
	h := crc32.New(crc32c.Table)
	_, err := io.Copy(h, r)
	must(err)
	return h.Sum(nil)
}

func fullCircle(t *testing.T, target string, source string) {
	sourceInfo, err := tlc.Walk(source, pwr.BlockSize)
	must(err)

	patch, err := ioutil.TempFile(os.TempDir(), "pwrtest")
	must(err)
	must(patch.Close())

	diff(target, source, patch.Name(), 1)

	tmpDir, err := ioutil.TempDir(os.TempDir(), "pwrtest")
	must(err)
	apply(patch.Name(), target, tmpDir)

	outputInfo, err := tlc.Walk(tmpDir, pwr.BlockSize)
	must(err)

	assert.Equal(t, sourceInfo, outputInfo, "must have recreated the same files!")

	b1 := contents(sourceInfo, source)
	b2 := contents(sourceInfo, tmpDir)
	assert.Equal(t, b1, b2, "must have the same contents")
}

func Test_Mega(t *testing.T) {
	*appArgs.no_progress = true
	fullCircle(t, "./fixtures/a", "./fixtures/b")
	fullCircle(t, "./fixtures/b", "./fixtures/a")
	fullCircle(t, ".", "./fixtures")
	fullCircle(t, "./fixtures", ".")
}
