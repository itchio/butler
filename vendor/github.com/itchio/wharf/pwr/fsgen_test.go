package pwr

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/alecthomas/assert"
)

var testSymlinks = (runtime.GOOS != "windows")

type testDirEntry struct {
	path string
	mode int
	size int64
	seed int64
	dir  bool
	dest string
}

type testDirSettings struct {
	seed    int64
	entries []testDirEntry
}

func makeTestDir(t *testing.T, dir string, s testDirSettings) {
	prng := rand.New(rand.NewSource(s.seed))

	assert.NoError(t, os.MkdirAll(dir, 0755))
	data := new(bytes.Buffer)

	for _, entry := range s.entries {
		path := filepath.Join(dir, filepath.FromSlash(entry.path))

		if entry.dir {
			mode := 0755
			if entry.mode != 0 {
				mode = entry.mode
			}
			assert.NoError(t, os.MkdirAll(entry.path, os.FileMode(mode)))
			continue
		} else if entry.dest != "" {
			assert.NoError(t, os.Symlink(entry.dest, path))
			continue
		}

		parent := filepath.Dir(path)
		mkErr := os.MkdirAll(parent, 0755)
		if mkErr != nil {
			if !os.IsExist(mkErr) {
				assert.NoError(t, mkErr)
			}
		}

		if entry.seed == 0 {
			prng.Seed(s.seed)
		} else {
			prng.Seed(entry.seed)
		}

		data.Reset()
		data.Grow(int(entry.size))

		func() {
			mode := 0644
			if entry.mode != 0 {
				mode = entry.mode
			}

			size := BlockSize*8 + 64
			if entry.size != 0 {
				size = entry.size
			}

			f, fErr := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.FileMode(mode))
			assert.NoError(t, fErr)
			defer f.Close()

			_, fErr = io.CopyN(f, prng, size)
			assert.NoError(t, fErr)
		}()
	}
}

func cpFile(t *testing.T, src string, dst string) {
	sf, fErr := os.Open(src)
	assert.NoError(t, fErr)
	defer sf.Close()

	info, fErr := sf.Stat()
	assert.NoError(t, fErr)

	df, fErr := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, info.Mode())
	assert.NoError(t, fErr)
	defer df.Close()

	_, fErr = io.Copy(df, sf)
	assert.NoError(t, fErr)
}

func cpDir(t *testing.T, src string, dst string) {
	assert.NoError(t, os.MkdirAll(dst, 0755))

	assert.NoError(t, filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		assert.NoError(t, err)
		name, fErr := filepath.Rel(src, path)
		assert.NoError(t, fErr)

		dstPath := filepath.Join(dst, name)

		if info.IsDir() {
			assert.NoError(t, os.MkdirAll(dstPath, info.Mode()))
		} else if info.Mode()&os.ModeSymlink > 0 {
			dest, fErr := os.Readlink(path)
			assert.NoError(t, fErr)

			assert.NoError(t, os.Symlink(dest, dstPath))
		} else if info.Mode().IsRegular() {
			df, fErr := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY, info.Mode())
			assert.NoError(t, fErr)
			defer df.Close()

			sf, fErr := os.Open(path)
			assert.NoError(t, fErr)
			defer sf.Close()

			_, fErr = io.Copy(df, sf)
			assert.NoError(t, fErr)
		} else {
			return fmt.Errorf("not regular, not symlink, not dir, what is it? %s", path)
		}

		return nil
	}))
}

func assertDirEmpty(t *testing.T, dir string) {
	files, err := ioutil.ReadDir(dir)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(files))
}
