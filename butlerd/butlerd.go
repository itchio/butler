package butlerd

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/itchio/headway/state"
	"github.com/pkg/errors"
)

type Server struct {
	secret string
}

func NewServer(secret string) *Server {
	return &Server{secret: secret}
}

type ServeTCPParams struct {
	Handler   jsonrpc2.Handler
	Consumer  *state.Consumer
	Listener  net.Listener
	Secret    string
	Log       bool
	KeepAlive bool

	ShutdownChan chan struct{}
}

func (s *Server) ServeTCP(ctx context.Context, params ServeTCPParams) error {
	if params.KeepAlive {
		return s.serveTCPKeepAlive(ctx, params)
	} else {
		return s.serveTCPClose(ctx, params)
	}
}

func (s *Server) serveTCPClose(ctx context.Context, params ServeTCPParams) error {
	tcpConn, err := params.Listener.Accept()
	if err != nil {
		return err
	}

	return s.handleTCPConn(ctx, params, tcpConn)
}

func (s *Server) serveTCPKeepAlive(ctx context.Context, params ServeTCPParams) error {
	var wg sync.WaitGroup
	conns := make(chan net.Conn)
	go func() {
		for {
			tcpConn, err := params.Listener.Accept()
			if err != nil {
				log.Printf("While accepting connection: %+v", err)
			}
			conns <- tcpConn
		}
	}()

	for {
		select {
		case tcpConn := <-conns:
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := s.handleTCPConn(ctx, params, tcpConn)
				if err != nil {
					log.Printf("While handling TCP connection: %+v", err)
				}
			}()
		case <-params.ShutdownChan:
			log.Printf("Closing TCP listener...")
			err := params.Listener.Close()
			if err != nil {
				log.Printf("While closing TCP listener: %+v", err)
			}

			log.Printf("Waiting for TCP connections to close...")
			wg.Wait()
			log.Printf("All TCP connections closed")

			return nil
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *Server) handleTCPConn(parentCtx context.Context, params ServeTCPParams, tcpConn net.Conn) error {
	gh := newGatedHandler(params.Handler, params.Secret)

	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	conn := jsonrpc2.NewConn(ctx, jsonrpc2.NewRwcTransport(tcpConn), gh)
	<-conn.DisconnectNotify()

	return nil
}

//

type gatedHandler struct {
	authenticateChan  chan struct{}
	authenticated     bool
	authenticateMutex sync.Mutex

	secret string
	inner  jsonrpc2.Handler
}

var _ jsonrpc2.Handler = (*gatedHandler)(nil)

func newGatedHandler(inner jsonrpc2.Handler, secret string) jsonrpc2.Handler {
	return &gatedHandler{
		authenticateChan: make(chan struct{}),
		authenticated:    false,

		secret: secret,
		inner:  inner,
	}
}

func (h *gatedHandler) HandleRequest(conn jsonrpc2.Conn, req jsonrpc2.Request) (interface{}, error) {
	if req.Method == "Meta.Authenticate" {
		var params MetaAuthenticateParams

		if req.Params != nil {
			err := jsonrpc2.DecodeJSON(*req.Params, &params)
			if err != nil {
				return nil, err
			}
		}

		if params.Secret != h.secret {
			return nil, errors.Errorf("Invalid secret")
		}

		func() {
			h.authenticateMutex.Lock()
			defer h.authenticateMutex.Unlock()

			if !h.authenticated {
				h.authenticated = true
				// notify any pending requests that they are free to go
				close(h.authenticateChan)
			}
		}()

		result := MetaAuthenticateResult{
			OK: true,
		}
		return result, nil
	} else {
		<-h.authenticateChan
		return h.inner.HandleRequest(conn, req)
	}
}

func (h *gatedHandler) HandleNotification(conn jsonrpc2.Conn, notif jsonrpc2.Notification) {
	h.inner.HandleNotification(conn, notif)
}
