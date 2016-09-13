package tlc

import (
	"io"
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/sync"
)

// ReadCloseSeeker unifies io.Reader, io.Seeker, and io.Closer
type ReadCloseSeeker interface {
	io.Reader
	io.Seeker
	io.Closer
}

// ContainerFilePool implements the sync.FilePool interface based on a Container
type ContainerFilePool struct {
	container *Container
	basePath  string

	fileIndex int64
	reader    ReadCloseSeeker
}

var _ sync.FilePool = (*ContainerFilePool)(nil)
var _ sync.WritableFilePool = (*ContainerFilePool)(nil)

// NewFilePool creates a new ContainerFilePool from the given Container
// metadata and a base path on-disk to allow reading from files.
func (c *Container) NewFilePool(basePath string) *ContainerFilePool {
	return &ContainerFilePool{
		container: c,
		basePath:  basePath,

		fileIndex: int64(-1),
		reader:    nil,
	}
}

// GetSize returns the size of the file at index fileIndex
func (cfp *ContainerFilePool) GetSize(fileIndex int64) int64 {
	return cfp.container.Files[fileIndex].Size
}

// GetRelativePath returns the slashed path of a file, relative to
// the container's root.
func (cfp *ContainerFilePool) GetRelativePath(fileIndex int64) string {
	return cfp.container.Files[fileIndex].Path
}

// GetPath returns the native path of a file (with slashes or backslashes)
// on-disk, based on the ContainerFilePool's base path
func (cfp *ContainerFilePool) GetPath(fileIndex int64) string {
	path := filepath.FromSlash(cfp.container.Files[fileIndex].Path)
	fullPath := filepath.Join(cfp.basePath, path)
	return fullPath
}

// GetReader returns an io.Reader for the file at index fileIndex
// Successive calls to `GetReader` will attempt to re-use the last
// returned reader if the file index is similar. The cache size is 1, so
// reading in parallel from different files is not supported.
func (cfp *ContainerFilePool) GetReader(fileIndex int64) (io.Reader, error) {
	return cfp.GetReadSeeker(fileIndex)
}

// GetReadSeeker is like GetReader but the returned object allows seeking
func (cfp *ContainerFilePool) GetReadSeeker(fileIndex int64) (io.ReadSeeker, error) {
	if cfp.fileIndex != fileIndex {
		if cfp.reader != nil {
			err := cfp.reader.Close()
			if err != nil {
				return nil, errors.Wrap(err, 1)
			}
			cfp.reader = nil
		}

		reader, err := os.Open(cfp.GetPath(fileIndex))

		if err != nil {
			return nil, errors.Wrap(err, 1)
		}
		cfp.reader = reader
		cfp.fileIndex = fileIndex
	}

	return cfp.reader, nil
}

// Close closes all reader belonging to this ContainerFilePool
func (cfp *ContainerFilePool) Close() error {
	if cfp.reader != nil {
		err := cfp.reader.Close()
		if err != nil {
			return errors.Wrap(err, 1)
		}

		cfp.reader = nil
		cfp.fileIndex = -1
	}

	return nil
}

func (cfp *ContainerFilePool) GetWriter(fileIndex int64) (io.WriteCloser, error) {
	path := cfp.GetPath(fileIndex)

	err := os.MkdirAll(filepath.Dir(path), os.FileMode(0755))
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	outputFile := cfp.container.Files[fileIndex]
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(outputFile.Mode)|ModeMask)
}
