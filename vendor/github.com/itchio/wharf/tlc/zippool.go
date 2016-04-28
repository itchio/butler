package tlc

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/itchio/wharf/sync"
)

// ContainerZipPool implements the sync.ZipPool interface based on a Container
type ContainerZipPool struct {
	container *Container
	fmap      map[string]*zip.File

	fileIndex int64
	reader    io.ReadCloser

	seekFileIndex int64
	readSeeker    ReadCloseSeeker
}

var _ sync.FilePool = (*ContainerZipPool)(nil)

// NewZipPool creates a new ContainerZipPool from the given Container
// metadata and a base path on-disk to allow reading from files.
func (c *Container) NewZipPool(zipReader *zip.Reader) *ContainerZipPool {
	fmap := make(map[string]*zip.File)
	for _, f := range zipReader.File {
		info := f.FileInfo()

		if info.IsDir() {
			// muffin
		} else if (info.Mode() & os.ModeSymlink) > 0 {
			// muffin ether
		} else {
			key := filepath.Clean(filepath.ToSlash(f.Name))
			fmap[key] = f
		}
	}

	return &ContainerZipPool{
		container: c,
		fmap:      fmap,

		fileIndex: int64(-1),
		reader:    nil,

		seekFileIndex: int64(-1),
		readSeeker:    nil,
	}
}

// GetSize returns the size of the file at index fileIndex
func (cfp *ContainerZipPool) GetSize(fileIndex int64) int64 {
	return cfp.container.Files[fileIndex].Size
}

// GetRelativePath returns the slashed path of a file, relative to
// the container's root.
func (cfp *ContainerZipPool) GetRelativePath(fileIndex int64) string {
	return cfp.container.Files[fileIndex].Path
}

// GetPath returns the native path of a file (with slashes or backslashes)
// on-disk, based on the ContainerZipPool's base path
func (cfp *ContainerZipPool) GetPath(fileIndex int64) string {
	panic("ContainerZipPool does not support GetPath")
}

// GetReader returns an io.Reader for the file at index fileIndex
// Successive calls to `GetReader` will attempt to re-use the last
// returned reader if the file index is similar. The cache size is 1, so
// reading in parallel from different files is not supported.
func (cfp *ContainerZipPool) GetReader(fileIndex int64) (io.Reader, error) {
	if cfp.fileIndex != fileIndex {
		if cfp.reader != nil {
			err := cfp.reader.Close()
			if err != nil {
				return nil, err
			}
			cfp.reader = nil
			cfp.fileIndex = -1
		}

		f := cfp.fmap[cfp.GetRelativePath(fileIndex)]
		if f == nil {
			return nil, os.ErrNotExist
		}

		reader, err := f.Open()

		if err != nil {
			return nil, err
		}
		cfp.reader = reader
		cfp.fileIndex = fileIndex
	}

	return cfp.reader, nil
}

// GetReadSeeker is like GetReader but the returned object allows seeking
func (cfp *ContainerZipPool) GetReadSeeker(fileIndex int64) (io.ReadSeeker, error) {
	if cfp.seekFileIndex != fileIndex {
		if cfp.readSeeker != nil {
			err := cfp.readSeeker.Close()
			if err != nil {
				return nil, err
			}
			cfp.readSeeker = nil
			cfp.seekFileIndex = -1
		}

		key := cfp.GetRelativePath(fileIndex)
		f := cfp.fmap[key]
		if f == nil {
			return nil, os.ErrNotExist
		}

		reader, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		buf, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		cfp.readSeeker = &closableBuf{bytes.NewReader(buf)}
		cfp.seekFileIndex = fileIndex
	}

	return cfp.readSeeker, nil
}

// Close closes all reader belonging to this ContainerZipPool
func (cfp *ContainerZipPool) Close() error {
	if cfp.reader != nil {
		err := cfp.reader.Close()
		if err != nil {
			return err
		}

		cfp.reader = nil
		cfp.fileIndex = -1
	}

	if cfp.readSeeker != nil {
		err := cfp.readSeeker.Close()
		if err != nil {
			return err
		}

		cfp.readSeeker = nil
		cfp.seekFileIndex = -1
	}

	return nil
}

type closableBuf struct {
	rs io.ReadSeeker
}

var _ ReadCloseSeeker = (*closableBuf)(nil)

func (cb *closableBuf) Read(buf []byte) (int, error) {
	return cb.rs.Read(buf)
}

func (cb *closableBuf) Seek(offset int64, whence int) (int64, error) {
	return cb.rs.Seek(offset, whence)
}

func (cb *closableBuf) Close() error {
	return nil
}
