package bfs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/itchio/lake/tlc"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type folderSpec struct {
	entries []*entrySpec
}

type entrySpec struct {
	name string
	data []byte
}

func cleanAndMakeFolder(fs *folderSpec, dest string) error {
	err := os.RemoveAll(dest)
	if err != nil {
		return errors.Wrap(err, "cleaning up test folder")
	}

	return makeFolder(fs, dest)
}

func makeFolder(fs *folderSpec, dest string) error {
	err := os.MkdirAll(dest, 0o755)
	if err != nil {
		return errors.Wrap(err, "creating test folder")
	}

	for _, e := range fs.entries {
		entryPath := filepath.Join(dest, e.name)
		entryDir := filepath.Dir(entryPath)

		err = os.MkdirAll(entryDir, 0o755)
		if err != nil {
			return errors.Wrap(err, "creating test folder directory entry")
		}

		err = ioutil.WriteFile(entryPath, e.data, os.FileMode(0o644))
		if err != nil {
			return errors.Wrap(err, "writing test folder file entry")
		}
	}

	return nil
}

func checkFolder(t *testing.T, fs *folderSpec, dest string) {
	entryNames := make(map[string]bool)
	for _, e := range fs.entries {
		entryNames[filepath.ToSlash(e.name)] = true

		entryPath := filepath.Join(dest, e.name)

		data, err := ioutil.ReadFile(entryPath)
		must(t, err)

		assert.EqualValues(t, e.data, data)
	}

	// make sure all entries are accounted for
	container, err := tlc.WalkDir(dest, &tlc.WalkOpts{
		Filter: tlc.DefaultFilter,
	})
	must(t, err)

	for _, f := range container.Files {
		if _, ok := entryNames[f.Path]; !ok {
			t.Errorf("extra file entry found: %s", f.Path)
		}
	}
}

func must(t *testing.T, err error) {
	if err != nil {
		assert.NoError(t, err)
		t.FailNow()
	}
}
