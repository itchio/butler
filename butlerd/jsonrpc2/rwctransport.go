package jsonrpc2

import (
	"bufio"
	"io"
)

type ReadWriteClose interface {
	io.Reader
	io.Writer
	io.Closer
}

type rwcTransport struct {
	inner   ReadWriteClose
	scanner *bufio.Scanner
	closed  bool
}

func NewRwcTransport(rwc ReadWriteClose) Transport {
	return &rwcTransport{
		inner:   rwc,
		scanner: bufio.NewScanner(rwc),
		closed:  false,
	}
}

func (rwc *rwcTransport) Read() ([]byte, error) {
	if rwc.closed {
		return nil, io.ErrClosedPipe
	}

	if rwc.scanner.Scan() {
		return rwc.scanner.Bytes(), nil
	}

	return nil, rwc.scanner.Err()
}

func (rwc *rwcTransport) Write(msg []byte) error {
	_, err := rwc.inner.Write(msg)
	if err != nil {
		return err
	}

	var separator = []byte{'\n'}

	_, err = rwc.inner.Write(separator)
	if err != nil {
		return err
	}
	return nil
}

func (rwc *rwcTransport) Close() error {
	if rwc.closed {
		return nil
	}

	rwc.closed = true
	return rwc.inner.Close()
}
