package main

import (
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/itchio/wharf.proto/megafile"
	"github.com/stretchr/testify/assert"
)

func contents(info *megafile.RepoInfo, path string) []byte {
	r := info.NewReader(path)
	b, err := ioutil.ReadAll(r)
	must(err)
	return b
}

func dirHash(info *megafile.RepoInfo, path string) []byte {
	r := info.NewReader(path)
	h := crc32.New(crc32cTable)
	_, err := io.Copy(h, r)
	must(err)
	return h.Sum(nil)
}

func Test_Mega(t *testing.T) {
	megafile.MEGAFILE_DEBUG = true
	*appArgs.no_progress = true

	target := "./fixtures/a"
	source := "./fixtures/b"

	sourceInfo, err := megafile.Walk(source, MP_BLOCK_SIZE)
	must(err)

	patch, err := ioutil.TempFile(os.TempDir(), "megatest")
	patch.Close()
	must(err)

	megadiff(target, source, patch.Name(), 1)

	tmpDir, err := ioutil.TempDir(os.TempDir(), "megatest")
	must(err)
	megapatch(patch.Name(), target, tmpDir)

	outputInfo, err := megafile.Walk(tmpDir, MP_BLOCK_SIZE)
	must(err)

	assert.Equal(t, sourceInfo, outputInfo, "must have recreated the same files!")

	b1 := contents(sourceInfo, source)
	b2 := contents(sourceInfo, tmpDir)
	assert.Equal(t, b1, b2, "must have the same contents")
}
