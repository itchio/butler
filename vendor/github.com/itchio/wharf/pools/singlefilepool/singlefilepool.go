package singlefilepool

import (
	"fmt"
	"io"

	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

type SingleFilePool struct {
	container *tlc.Container
	wr        io.Writer
}

var _ wsync.WritablePool = (*SingleFilePool)(nil)

func New(container *tlc.Container, wr io.Writer) *SingleFilePool {
	return &SingleFilePool{
		container: container,
		wr:        wr,
	}
}

func (sfp *SingleFilePool) GetSize(fileIndex int64) int64 {
	return 0
}

func (sfp *SingleFilePool) GetReader(fileIndex int64) (io.Reader, error) {
	return nil, fmt.Errorf("SingleFilePool is not readable")
}

func (sfp *SingleFilePool) GetReadSeeker(fileIndex int64) (io.ReadSeeker, error) {
	return nil, fmt.Errorf("SingleFilePool is not readable")
}

func (sfp *SingleFilePool) GetWriter(fileIndex int64) (io.WriteCloser, error) {
	return &nopWriteCloser{sfp.wr}, nil
}

func (sfp *SingleFilePool) Close() error {
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
