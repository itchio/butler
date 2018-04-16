package bfs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Walk(t *testing.T) {
	dir, err := ioutil.TempDir("", "bfs-walk-test")
	must(t, err)
	defer func() {
		os.RemoveAll(dir)
	}()

	dest := filepath.Join(dir, "some-dir")
	fs := &folderSpec{
		entries: []*entrySpec{
			{
				name: "a/deep/very/deep/thing.txt",
				data: []byte("Hi"),
			},
			{
				name: "a/deep/thingamaji.txt",
				data: []byte("Oh well"),
			},
			{
				name: "shallow.txt",
				data: []byte("Really?"),
			},
			{
				name: ".itch/receipt.json",
				data: []byte("{\"hello\":\"world\"}"),
			},
			{
				name: ".itch/receipt.json.gz",
				data: []byte{0x1f, 0x8b /* this is fake btw */, 0x01, 0x02, 0x04},
			},
		},
	}

	must(t, makeFolder(fs, dest))
	container, err := Walk(dest)
	assert.NoError(t, err)

	assert.EqualValues(t, 3, len(container.Files))
}
