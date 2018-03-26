package zipwriterpool

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/itchio/arkive/zip"

	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
)

type ZipWriterPool struct {
	container *tlc.Container
	zw        *zip.Writer
}

var _ wsync.WritablePool = (*ZipWriterPool)(nil)

func New(container *tlc.Container, zw *zip.Writer) *ZipWriterPool {
	return &ZipWriterPool{
		container: container,
		zw:        zw,
	}
}

func (zwp *ZipWriterPool) GetSize(fileIndex int64) int64 {
	return 0
}

func (zwp *ZipWriterPool) GetReader(fileIndex int64) (io.Reader, error) {
	return nil, fmt.Errorf("zipwriterpool is not readable")
}

func (zwp *ZipWriterPool) GetReadSeeker(fileIndex int64) (io.ReadSeeker, error) {
	return nil, fmt.Errorf("zipwriterpool is not readable")
}

func (zwp *ZipWriterPool) GetWriter(fileIndex int64) (io.WriteCloser, error) {
	file := zwp.container.Files[fileIndex]

	fh := zip.FileHeader{
		Name:               file.Path,
		UncompressedSize64: uint64(file.Size),
		Method:             zip.Deflate,
	}
	fh.SetMode(os.FileMode(file.Mode))
	fh.SetModTime(time.Now())

	w, err := zwp.zw.CreateHeader(&fh)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &nopWriteCloser{w}, nil
}

// Close writes symlinks and dirs of the container, then closes
// the zip writer.
func (zwp *ZipWriterPool) Close() error {
	for _, symlink := range zwp.container.Symlinks {
		fh := zip.FileHeader{
			Name: symlink.Path,
		}
		fh.SetMode(os.FileMode(symlink.Mode))

		entryWriter, eErr := zwp.zw.CreateHeader(&fh)
		if eErr != nil {
			return errors.WithStack(eErr)
		}

		entryWriter.Write([]byte(symlink.Dest))
	}

	for _, dir := range zwp.container.Dirs {
		fh := zip.FileHeader{
			Name: dir.Path + "/",
		}
		fh.SetMode(os.FileMode(dir.Mode))
		fh.SetModTime(time.Now())

		_, hErr := zwp.zw.CreateHeader(&fh)
		if hErr != nil {
			return errors.WithStack(hErr)
		}
	}

	err := zwp.zw.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// nopWriteCloser

type nopWriteCloser struct {
	writer io.Writer
}

var _ io.Writer = (*nopWriteCloser)(nil)

func (nwc *nopWriteCloser) Write(data []byte) (int, error) {
	return nwc.writer.Write(data)
}

func (nwc *nopWriteCloser) Close() error {
	return nil
}
