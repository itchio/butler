package nullpool

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

type NullPool struct {
	container *tlc.Container
}

var _ wsync.Pool = (*NullPool)(nil)
var _ wsync.WritablePool = (*NullPool)(nil)

func New(container *tlc.Container) *NullPool {
	return &NullPool{container}
}

func (fp *NullPool) GetSize(fileIndex int64) int64 {
	return 0
}

func (fp *NullPool) GetReader(fileIndex int64) (io.Reader, error) {
	return fp.GetReadSeeker(fileIndex)
}

func (fp *NullPool) GetReadSeeker(fileIndex int64) (io.ReadSeeker, error) {
	return &NullReader{
		size: fp.container.Files[fileIndex].Size,
	}, nil
}

type NullReader struct {
	offset int64
	size   int64
}

func (nr *NullReader) Read(buf []byte) (int, error) {
	newOffset := nr.offset + int64(len(buf))
	if newOffset >= nr.size {
		newOffset = nr.size
	}

	readSize := int(newOffset - nr.offset)
	nr.offset = newOffset

	if readSize == 0 {
		return 0, io.EOF
	} else {
		return readSize, nil
	}
}

func (nr *NullReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case os.SEEK_END:
		nr.offset = nr.size + offset
	case os.SEEK_CUR:
		nr.offset += offset
	case os.SEEK_SET:
		nr.offset = offset
	}
	return nr.offset, nil
}

func (fp *NullPool) Close() error {
	return nil
}

type NopWriteCloser struct {
	writer io.Writer
}

func (nwc *NopWriteCloser) Write(buf []byte) (int, error) {
	return nwc.writer.Write(buf)
}

func (nwc *NopWriteCloser) Close() error {
	return nil
}

func (fp *NullPool) GetWriter(fileIndex int64) (io.WriteCloser, error) {
	return &NopWriteCloser{ioutil.Discard}, nil
}
