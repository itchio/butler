package bfs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/itchio/headway/state"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_SaveAngels(t *testing.T) {
	dir, err := ioutil.TempDir("", "bfs-save-angels-test")
	must(t, err)
	defer func() {
		os.RemoveAll(dir)
	}()

	angelEntry := &entrySpec{
		name: "hey/here/is/a/giant-save.dat",
		data: []byte{0xfe, 0xf1, 0xf0, 0xf8},
	}

	angelEntry2 := &entrySpec{
		name: "mods/enable-all.pvk",
		data: []byte("fancy drm, here my shout"),
	}

	dest := filepath.Join(dir, "sample-app")
	oldFs := &folderSpec{
		entries: []*entrySpec{
			{
				name: "text/shakespeare2000.txt",
				data: []byte("There are more things in network and disk, Horatio,\nThan are dreamt of in your codebase."),
			},
			{
				name: ".gitkeep",
				data: []byte{},
			},
			angelEntry,
			angelEntry2,
		},
	}

	sampleReceipt := &Receipt{
		// don't fill other fields of Receipt, that's fine
		Files: []string{
			"text/shakespeare2000.txt",
			".gitkeep",
		},
	}

	newEntry := &entrySpec{
		name: "text/shakespeare-commentary.txt",
		data: []byte("Shakespeare does not expand on the specific nature of Horatio's codebase, [...]"),
	}

	newFs := &folderSpec{
		entries: []*entrySpec{
			newEntry,
		},
	}

	successResult := &SaveAngelsResult{
		Files: []string{
			newEntry.name,
		},
	}

	newFsWithAngels := &folderSpec{
		entries: []*entrySpec{
			newEntry,
			angelEntry,
			angelEntry2,
		},
	}

	params := &SaveAngelsParams{
		Consumer: &state.Consumer{
			OnMessage: func(lvl string, msg string) {
				t.Logf("[%s] %s", lvl, msg)
			},
		},
		Folder:  dest,
		Receipt: nil,
	}

	taskCalled := false

	succeedingTask := func() error {
		taskCalled = true
		must(t, makeFolder(newFs, dest))
		return nil
	}

	taskFailedErr := errors.New("uh oh the task failed")

	failingTask := func() error {
		taskCalled = true
		must(t, makeFolder(newFs, dest))
		return taskFailedErr
	}

	{
		t.Logf("Succeding task, no receipt, no old dir")
		taskCalled = false
		must(t, os.RemoveAll(dest))
		result, err := SaveAngels(params, succeedingTask)
		assert.NoError(t, err)
		assertEqualResult(t, successResult, result)
		assert.True(t, taskCalled)
		checkFolder(t, newFs, dest)
	}

	{
		t.Logf("Failing task, no receipt, no old dir")
		taskCalled = false
		must(t, os.RemoveAll(dest))
		_, err := SaveAngels(params, failingTask)
		assert.Error(t, err)
		assert.True(t, errors.Cause(err) == taskFailedErr)
		assert.True(t, taskCalled)
	}

	{
		t.Logf("Succeding task, no receipt")
		taskCalled = false
		must(t, cleanAndMakeFolder(oldFs, dest))
		result, err := SaveAngels(params, succeedingTask)
		assert.NoError(t, err)
		assertEqualResult(t, successResult, result)
		assert.True(t, taskCalled)
		checkFolder(t, newFs, dest)
	}

	{
		t.Logf("Failing task, no receipt")
		taskCalled = false
		must(t, cleanAndMakeFolder(oldFs, dest))
		_, err := SaveAngels(params, failingTask)
		assert.Error(t, err)
		assert.True(t, errors.Cause(err) == taskFailedErr)
		assert.True(t, taskCalled)
		checkFolder(t, oldFs, dest)
	}

	params.Receipt = sampleReceipt

	{
		t.Logf("Succeding task, with receipt")
		taskCalled = false
		must(t, cleanAndMakeFolder(oldFs, dest))
		result, err := SaveAngels(params, succeedingTask)
		assert.NoError(t, err)
		// it's important that angels doesn't make it into the result
		assertEqualResult(t, successResult, result)

		checkFolder(t, newFsWithAngels, dest)
	}

	{
		t.Logf("Failing task, with receipt")
		taskCalled = false
		must(t, cleanAndMakeFolder(oldFs, dest))
		_, err := SaveAngels(params, failingTask)
		assert.Error(t, err)
		assert.True(t, errors.Cause(err) == taskFailedErr)
		assert.True(t, taskCalled)
		checkFolder(t, oldFs, dest)
	}
}

func assertEqualResult(t *testing.T, expected *SaveAngelsResult, actual *SaveAngelsResult) {
	sort.Strings(expected.Files)
	sort.Strings(actual.Files)

	assert.EqualValues(t, expected, actual)
}
