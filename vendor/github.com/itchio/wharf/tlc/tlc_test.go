package tlc_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/itchio/wharf/tlc"
	"github.com/stretchr/testify/assert"
)

func Test_Walk(t *testing.T) {
	tmpPath := mktestdir(t, "walk")
	defer func() {
		err := os.RemoveAll(tmpPath)
		must(t, err)
	}()

	info, err := tlc.Walk(tmpPath, nil)
	must(t, err)

	dirs := []string{
		".",
		"foo",
		"foo/dir_a",
		"foo/dir_b",
	}
	for i, dir := range dirs {
		assert.Equal(t, dir, info.Dirs[i].Path, "dirs should be all listed")
	}

	files := []string{
		"foo/dir_a/baz",
		"foo/dir_a/bazzz",
		"foo/dir_b/zoom",
		"foo/file_f",
		"foo/file_z",
	}
	for i, file := range files {
		assert.Equal(t, file, info.Files[i].Path, "files should be all listed")
	}

	if testSymlinks {
		for i, symlink := range symlinks {
			assert.Equal(t, symlink.Newname, info.Symlinks[i].Path, "symlink should be at correct path")
			assert.Equal(t, symlink.Oldname, info.Symlinks[i].Dest, "symlink should point to correct path")
		}
	}
}

func Test_Prepare(t *testing.T) {
	tmpPath := mktestdir(t, "prepare")
	defer func() {
		err := os.RemoveAll(tmpPath)
		must(t, err)
	}()

	info, err := tlc.Walk(tmpPath, nil)
	must(t, err)

	tmpPath2, err := ioutil.TempDir(".", "tmp_prepare")
	must(t, err)

	err = info.Prepare(tmpPath2)
	must(t, err)

	info2, err := tlc.Walk(tmpPath2, nil)
	must(t, err)
	assert.Equal(t, info, info2, "must recreate same structure")
}

// Support code

func must(t *testing.T, err error) {
	if err != nil {
		t.Error("must failed: ", err.Error())
		t.FailNow()
	}
}

type regEntry struct {
	Path string
	Size int
	Byte byte
}

type symlinkEntry struct {
	Oldname string
	Newname string
}

var regulars = []regEntry{
	{"foo/file_f", 50, 0xd},
	{"foo/dir_a/baz", 10, 0xa},
	{"foo/dir_b/zoom", 30, 0xc},
	{"foo/file_z", 40, 0xe},
	{"foo/dir_a/bazzz", 20, 0xb},
}

var symlinks = []symlinkEntry{
	{"file_z", "foo/file_m"},
	{"dir_a/baz", "foo/file_o"},
}

var testSymlinks = runtime.GOOS != "windows"

func mktestdir(t *testing.T, name string) string {
	tmpPath, err := ioutil.TempDir(".", "tmp_"+name)
	must(t, err)

	must(t, os.RemoveAll(tmpPath))

	for _, entry := range regulars {
		fullPath := filepath.Join(tmpPath, entry.Path)
		must(t, os.MkdirAll(filepath.Dir(fullPath), os.FileMode(0777)))
		file, err := os.Create(fullPath)
		must(t, err)

		filler := []byte{entry.Byte}
		for i := 0; i < entry.Size; i++ {
			_, err := file.Write(filler)
			must(t, err)
		}
		must(t, file.Close())
	}

	if testSymlinks {
		for _, entry := range symlinks {
			new := filepath.Join(tmpPath, entry.Newname)
			must(t, os.Symlink(entry.Oldname, new))
		}
	}

	return tmpPath
}
