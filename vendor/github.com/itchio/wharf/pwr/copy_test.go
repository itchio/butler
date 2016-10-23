package pwr

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/alecthomas/assert"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
)

func Test_CopyContainer(t *testing.T) {
	mainDir, err := ioutil.TempDir("", "copycontainer")
	assert.NoError(t, err)
	defer os.RemoveAll(mainDir)

	src := path.Join(mainDir, "src")
	dst := path.Join(mainDir, "dst")
	makeTestDir(t, src, testDirSettings{
		seed: 0x91738,
		entries: []testDirEntry{
			{path: "subdir/file-1", seed: 0x1},
			{path: "file-1", seed: 0x2},
			{path: "file-2", seed: 0x3},
		},
	})

	container, err := tlc.WalkAny(src, nil)
	assert.NoError(t, err)

	inPool := fspool.New(container, src)
	outPool := fspool.New(container, dst)

	err = CopyContainer(container, outPool, inPool, &state.Consumer{})
	assert.NoError(t, err)

	assert.NoError(t, inPool.Close())
}
