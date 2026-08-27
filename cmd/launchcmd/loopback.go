package launchcmd

import (
	"io"
	"sync"

	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/pkg/errors"
)

// loopbackPipe returns two crossed in-memory jsonrpc2 transports, so a
// router and a client can talk in-process without a socket or secret.
func loopbackPipe() (jsonrpc2.Transport, jsonrpc2.Transport) {
	aToB := make(chan []byte, 16)
	bToA := make(chan []byte, 16)
	closed := make(chan struct{})
	var once sync.Once
	closeBoth := func() { once.Do(func() { close(closed) }) }

	a := &loopbackTransport{in: bToA, out: aToB, closed: closed, close: closeBoth}
	b := &loopbackTransport{in: aToB, out: bToA, closed: closed, close: closeBoth}
	return a, b
}

type loopbackTransport struct {
	in     chan []byte
	out    chan []byte
	closed chan struct{}
	close  func()
}

func (t *loopbackTransport) Read() ([]byte, error) {
	select {
	case msg := <-t.in:
		return msg, nil
	case <-t.closed:
		return nil, io.EOF
	}
}

func (t *loopbackTransport) Write(msg []byte) error {
	select {
	case t.out <- msg:
		return nil
	case <-t.closed:
		return errors.New("loopback transport closed")
	}
}

func (t *loopbackTransport) Close() error {
	t.close()
	return nil
}
