package wtest

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

	"github.com/itchio/wharf/pwr/drip"
	"github.com/itchio/wharf/wrand"
	"github.com/stretchr/testify/assert"
)

// see pwr/constants - the same BlockSize is used here to
// generate test data that diffs in a certain way
const BlockSize int64 = 64 * 1024 // 64k

var TestSymlinks = (runtime.GOOS != "windows")

type TestDirEntry struct {
	Path      string
	Mode      int
	Size      int64
	Seed      int64
	Dir       bool
	Dest      string
	Chunks    []TestDirChunk
	Bsmods    []Bsmod
	Swaperoos []Swaperoo
	Data      []byte
}

// Swaperoo swaps two blocks of the file
type Swaperoo struct {
	OldStart int64
	NewStart int64
	Size     int64
}

// Bsmode represents a bsdiff-like corruption
type Bsmod struct {
	// corrupt one byte every `interval`
	Interval int64

	// how much to add to the byte being corrupted
	Delta byte

	// only corrupt `max` times at a time, then skip `skip*interval` bytes
	Max  int
	Skip int
}

type TestDirChunk struct {
	Seed int64
	Size int64
}

type TestDirSettings struct {
	Seed    int64
	Entries []TestDirEntry
}

func MakeTestDir(t *testing.T, dir string, s TestDirSettings) {
	prng := wrand.RandReader{
		Source: rand.New(rand.NewSource(s.Seed)),
	}

	Must(t, os.MkdirAll(dir, 0755))
	data := new(bytes.Buffer)

	for _, entry := range s.Entries {
		path := filepath.Join(dir, filepath.FromSlash(entry.Path))

		if entry.Dir {
			mode := 0755
			if entry.Mode != 0 {
				mode = entry.Mode
			}
			Must(t, os.MkdirAll(entry.Path, os.FileMode(mode)))
			continue
		} else if entry.Dest != "" {
			Must(t, os.Symlink(entry.Dest, path))
			continue
		}

		parent := filepath.Dir(path)
		mkErr := os.MkdirAll(parent, 0755)
		if mkErr != nil {
			if !os.IsExist(mkErr) {
				Must(t, mkErr)
			}
		}

		if entry.Seed == 0 {
			prng.Seed(s.Seed)
		} else {
			prng.Seed(entry.Seed)
		}

		data.Reset()
		data.Grow(int(entry.Size))

		func() {
			mode := 0644
			if entry.Mode != 0 {
				mode = entry.Mode
			}

			size := BlockSize*8 + 64
			if entry.Size != 0 {
				size = entry.Size
			}

			f := new(bytes.Buffer)
			var err error

			if entry.Data != nil {
				_, err = f.Write(entry.Data)
				Must(t, err)
			} else if len(entry.Chunks) > 0 {
				for _, chunk := range entry.Chunks {
					prng.Seed(chunk.Seed)
					data.Reset()
					data.Grow(int(chunk.Size))

					_, err = io.CopyN(f, prng, chunk.Size)
					Must(t, err)
				}
			} else if len(entry.Bsmods) > 0 {
				func() {
					var writer io.Writer = NopWriteCloser(f)
					for _, bsmod := range entry.Bsmods {
						modcount := 0
						skipcount := 0

						drip := &drip.Writer{
							Buffer: make([]byte, bsmod.Interval),
							Writer: writer,
							Validate: func(data []byte) error {
								if bsmod.Max > 0 && modcount >= bsmod.Max {
									skipcount = bsmod.Skip
									modcount = 0
								}

								if skipcount > 0 {
									skipcount--
									return nil
								}

								data[0] = data[0] + bsmod.Delta
								modcount++
								return nil
							},
						}
						defer drip.Close()
						writer = drip
					}

					_, err = io.CopyN(writer, prng, size)
					Must(t, err)
				}()
			} else {
				_, err = io.CopyN(f, prng, size)
				Must(t, err)
			}

			finalBuf := f.Bytes()
			for _, s := range entry.Swaperoos {
				stagingBuf := make([]byte, s.Size)
				copy(stagingBuf, finalBuf[s.OldStart:s.OldStart+s.Size])
				copy(finalBuf[s.OldStart:s.OldStart+s.Size], finalBuf[s.NewStart:s.NewStart+s.Size])
				copy(finalBuf[s.NewStart:s.NewStart+s.Size], stagingBuf)
			}

			err = ioutil.WriteFile(path, finalBuf, os.FileMode(mode))
			Must(t, err)
		}()
	}
}

func CpFile(t *testing.T, src string, dst string) {
	sf, fErr := os.Open(src)
	Must(t, fErr)
	defer sf.Close()

	info, fErr := sf.Stat()
	Must(t, fErr)

	df, fErr := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, info.Mode())
	Must(t, fErr)
	defer df.Close()

	_, fErr = io.Copy(df, sf)
	Must(t, fErr)
}

func CpDir(t *testing.T, src string, dst string) {
	Must(t, os.MkdirAll(dst, 0755))

	Must(t, filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		Must(t, err)
		name, fErr := filepath.Rel(src, path)
		Must(t, fErr)

		dstPath := filepath.Join(dst, name)

		if info.IsDir() {
			Must(t, os.MkdirAll(dstPath, info.Mode()))
		} else if info.Mode()&os.ModeSymlink > 0 {
			dest, fErr := os.Readlink(path)
			Must(t, fErr)

			Must(t, os.Symlink(dest, dstPath))
		} else if info.Mode().IsRegular() {
			df, fErr := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY, info.Mode())
			Must(t, fErr)
			defer df.Close()

			sf, fErr := os.Open(path)
			Must(t, fErr)
			defer sf.Close()

			_, fErr = io.Copy(df, sf)
			Must(t, fErr)
		} else {
			return fmt.Errorf("not regular, not symlink, not dir, what is it? %s", path)
		}

		return nil
	}))
}

func AssertDirEmpty(t *testing.T, dir string) {
	files, err := ioutil.ReadDir(dir)
	Must(t, err)
	assert.Equal(t, 0, len(files))
}
