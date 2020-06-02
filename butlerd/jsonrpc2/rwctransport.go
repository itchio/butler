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
		return nil, io.EOF
	}

	if rwc.scanner.Scan() {
		bs := rwc.scanner.Bytes()
		return bs, nil
	}

	err := rwc.scanner.Err()
	if err != nil {
		return nil, err
	}

	return nil, io.EOF
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
