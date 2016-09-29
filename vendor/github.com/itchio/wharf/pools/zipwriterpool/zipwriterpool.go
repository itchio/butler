package zipwriterpool

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
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
		return nil, errors.Wrap(err, 1)
	}

	return &nopWriteCloser{w}, nil
}

func (zwp *ZipWriterPool) Close() error {
	err := zwp.zw.Close()
	if err != nil {
		return errors.Wrap(err, 1)
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
