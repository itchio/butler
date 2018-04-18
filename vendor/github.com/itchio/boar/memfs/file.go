package memfs

import (
	"bytes"
	"os"
	"time"

	"github.com/itchio/wharf/eos"
)

// New returns an eos.File with the given data and name
func New(data []byte, name string) eos.File {
	return &memoryFile{
		br:   bytes.NewReader(data),
		name: name,
	}
}

type memoryFile struct {
	br   *bytes.Reader
	name string
}

var _ eos.File = (*memoryFile)(nil)

func (mf *memoryFile) Close() error {
	// all hail the all-seeing eye of the GC
	return nil
}

func (mf *memoryFile) Read(buf []byte) (int, error) {
	return mf.br.Read(buf)
}

func (mf *memoryFile) ReadAt(buf []byte, offset int64) (int, error) {
	return mf.br.ReadAt(buf, offset)
}

func (mf *memoryFile) Seek(offset int64, whence int) (int64, error) {
	return mf.br.Seek(offset, whence)
}

func (mf *memoryFile) Stat() (os.FileInfo, error) {
	return &memoryFileInfo{mf}, nil
}

// memoryFileInfo implements os.FileInfo for memoryfiles
type memoryFileInfo struct {
	mf *memoryFile
}

var _ os.FileInfo = (*memoryFileInfo)(nil)

func (mfi *memoryFileInfo) Name() string {
	return mfi.mf.name
}

func (mfi *memoryFileInfo) Size() int64 {
	return mfi.mf.br.Size()
}

func (mfi *memoryFileInfo) Mode() os.FileMode {
	return os.FileMode(0)
}

func (mfi *memoryFileInfo) ModTime() time.Time {
	return time.Now()
}

func (mfi *memoryFileInfo) IsDir() bool {
	return false
}

func (mfi *memoryFileInfo) Sys() interface{} {
	return nil
}
