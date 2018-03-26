package zippool

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/itchio/arkive/zip"

	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
)

// ZipPool implements the wsync.ZipPool interface based on a Container
type ZipPool struct {
	container *tlc.Container
	fmap      map[string]*zip.File

	fileIndex int64
	reader    io.ReadCloser

	seekFileIndex int64
	readSeeker    ReadCloseSeeker
}

var _ wsync.Pool = (*ZipPool)(nil)

// ReadCloseSeeker unifies io.Reader, io.Seeker, and io.Closer
type ReadCloseSeeker interface {
	io.Reader
	io.Seeker
	io.Closer
}

// NewZipPool creates a new ZipPool from the given Container
// metadata and a base path on-disk to allow reading from files.
func New(c *tlc.Container, zipReader *zip.Reader) *ZipPool {
	fmap := make(map[string]*zip.File)
	for _, f := range zipReader.File {
		info := f.FileInfo()

		if info.IsDir() {
			// muffin
		} else if (info.Mode() & os.ModeSymlink) > 0 {
			// muffin ether
		} else {
			key := filepath.ToSlash(filepath.Clean(f.Name))
			fmap[key] = f
		}
	}

	return &ZipPool{
		container: c,
		fmap:      fmap,

		fileIndex: int64(-1),
		reader:    nil,

		seekFileIndex: int64(-1),
		readSeeker:    nil,
	}
}

// GetSize returns the size of the file at index fileIndex
func (cfp *ZipPool) GetSize(fileIndex int64) int64 {
	return cfp.container.Files[fileIndex].Size
}

// GetRelativePath returns the slashed path of a file, relative to
// the container's root.
func (cfp *ZipPool) GetRelativePath(fileIndex int64) string {
	return cfp.container.Files[fileIndex].Path
}

// GetPath returns the native path of a file (with slashes or backslashes)
// on-disk, based on the ZipPool's base path
func (cfp *ZipPool) GetPath(fileIndex int64) string {
	panic("ZipPool does not support GetPath")
}

// GetReader returns an io.Reader for the file at index fileIndex
// Successive calls to `GetReader` will attempt to re-use the last
// returned reader if the file index is similar. The cache size is 1, so
// reading in parallel from different files is not supported.
func (cfp *ZipPool) GetReader(fileIndex int64) (io.Reader, error) {
	if cfp.fileIndex != fileIndex {
		if cfp.reader != nil {
			err := cfp.reader.Close()
			if err != nil {
				return nil, errors.WithStack(err)
			}
			cfp.reader = nil
			cfp.fileIndex = -1
		}

		relPath := cfp.GetRelativePath(fileIndex)
		f := cfp.fmap[relPath]
		if f == nil {
			if os.Getenv("VERBOSE_ZIP_POOL") != "" {
				fmt.Printf("\nzip contents:\n")
				for k := range cfp.fmap {
					fmt.Printf("\n- %s", k)
				}
				fmt.Println()
			}
			return nil, errors.Wrap(os.ErrNotExist, relPath)
		}

		reader, err := f.Open()

		if err != nil {
			return nil, errors.WithStack(err)
		}
		cfp.reader = reader
		cfp.fileIndex = fileIndex
	}

	return cfp.reader, nil
}

// GetReadSeeker is like GetReader but the returned object allows seeking
func (cfp *ZipPool) GetReadSeeker(fileIndex int64) (io.ReadSeeker, error) {
	if cfp.seekFileIndex != fileIndex {
		if cfp.readSeeker != nil {
			err := cfp.readSeeker.Close()
			if err != nil {
				return nil, errors.WithStack(err)
			}
			cfp.readSeeker = nil
			cfp.seekFileIndex = -1
		}

		key := cfp.GetRelativePath(fileIndex)
		f := cfp.fmap[key]
		if f == nil {
			return nil, errors.WithStack(os.ErrNotExist)
		}

		reader, err := f.Open()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		defer reader.Close()

		buf, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		cfp.readSeeker = &closableBuf{bytes.NewReader(buf)}
		cfp.seekFileIndex = fileIndex
	}

	return cfp.readSeeker, nil
}

// Close closes all reader belonging to this ZipPool
func (cfp *ZipPool) Close() error {
	if cfp.reader != nil {
		err := cfp.reader.Close()
		if err != nil {
			return errors.WithStack(err)
		}

		cfp.reader = nil
		cfp.fileIndex = -1
	}

	if cfp.readSeeker != nil {
		err := cfp.readSeeker.Close()
		if err != nil {
			return errors.WithStack(err)
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
