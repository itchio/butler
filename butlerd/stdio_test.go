package butlerd

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/itchio/butler/butlerd/jsonrpc2"
)

type testHandler struct {
	handleRequest      func(conn jsonrpc2.Conn, req jsonrpc2.Request) (interface{}, error)
	handleNotification func(conn jsonrpc2.Conn, notif jsonrpc2.Notification)
}

var _ jsonrpc2.Handler = (*testHandler)(nil)

func (h *testHandler) HandleRequest(conn jsonrpc2.Conn, req jsonrpc2.Request) (interface{}, error) {
	if h.handleRequest != nil {
		return h.handleRequest(conn, req)
	}
	return nil, errors.New("unexpected request")
}

func (h *testHandler) HandleNotification(conn jsonrpc2.Conn, notif jsonrpc2.Notification) {
	if h.handleNotification != nil {
		h.handleNotification(conn, notif)
	}
}

func Test_ServeStdio_AllowsRequestsWithoutMetaAuthenticate(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	shutdownChan := make(chan struct{})
	s := NewServer("secret-unused-for-stdio")

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- s.ServeStdio(context.Background(), ServeStdioParams{
			Handler: &testHandler{
				handleRequest: func(conn jsonrpc2.Conn, req jsonrpc2.Request) (interface{}, error) {
					if req.Method != "Ping" {
						return nil, errors.New("unexpected method")
					}
					return map[string]bool{"ok": true}, nil
				},
			},
			ShutdownChan: shutdownChan,
		}, serverConn)
	}()

	client := jsonrpc2.NewConn(context.Background(), jsonrpc2.NewRwcTransport(clientConn), &testHandler{})
	defer client.Close()

	var result struct {
		OK bool `json:"ok"`
	}

	err := client.Call("Ping", struct{}{}, &result)
	if err != nil {
		t.Fatalf("calling Ping over stdio: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got false")
	}

	close(shutdownChan)
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("ServeStdio returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for ServeStdio to return after shutdown")
	}
}

func Test_ServeStdio_StopsWhenShutdownSignaled(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	shutdownChan := make(chan struct{})
	s := NewServer("secret-unused-for-stdio")

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- s.ServeStdio(context.Background(), ServeStdioParams{
			Handler:      &testHandler{},
			ShutdownChan: shutdownChan,
		}, serverConn)
	}()

	close(shutdownChan)
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("ServeStdio returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for ServeStdio to return after shutdown")
	}
}

func Test_ServeStdio_StopsOnClientDisconnect(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	shutdownChan := make(chan struct{})
	s := NewServer("secret-unused-for-stdio")

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- s.ServeStdio(context.Background(), ServeStdioParams{
			Handler:      &testHandler{},
			ShutdownChan: shutdownChan,
		}, serverConn)
	}()

	if err := clientConn.Close(); err != nil {
		t.Fatalf("closing client connection: %v", err)
	}

	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("ServeStdio returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for ServeStdio to return after client disconnect")
	}
}
