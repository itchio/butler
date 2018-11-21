package butlerd

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/handlers"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"
)

type Server struct {
	secret string
}

func NewServer(secret string) *Server {
	return &Server{secret: secret}
}

type ServeHTTPParams struct {
	HTTPListener net.Listener

	HTTPSListener net.Listener
	TLSState      *TLSState

	Handler  jsonrpc2.Handler
	Consumer *state.Consumer

	ShutdownChan chan struct{}

	Log bool
}

func (s *Server) ServeHTTP(ctx context.Context, params ServeHTTPParams) error {
	var shutdownWaitGroup sync.WaitGroup

	handleGracefulShutdown := func(srv *http.Server, name string) {
		go func() {
			select {
			case <-ctx.Done():
				// welp
			case <-params.ShutdownChan:
				shutdownWaitGroup.Add(1)
				defer shutdownWaitGroup.Done()
				log.Printf("Shutting down %s server gracefully...", name)
				err := srv.Shutdown(ctx)
				if err != nil {
					log.Printf("While performing %s server shutdown: %+v", name, err)
				}
				log.Printf("%s server has shut down.", name)
			}
		}()
	}

	hh := &httpHandler{
		jrh:    params.Handler,
		secret: s.secret,
	}

	var chosenHandler http.Handler = hh
	if params.Log {
		chosenHandler = handlers.LoggingHandler(os.Stderr, chosenHandler)
	}

	errs := make(chan error)
	go func() {
		tlsListener := tls.NewListener(params.HTTPSListener, params.TLSState.Config)
		srv := &http.Server{Handler: chosenHandler}
		srv.TLSConfig = params.TLSState.Config
		handleGracefulShutdown(srv, "https")
		errs <- srv.Serve(tlsListener)
	}()

	go func() {
		srv := &http.Server{Handler: chosenHandler}
		handleGracefulShutdown(srv, "http")
		errs <- srv.Serve(params.HTTPListener)
	}()

	for i := 0; i < 2; i++ {
		err := <-errs
		if err != nil {
			if errors.Cause(err) == http.ErrServerClosed {
				// that's ok!
				continue
			}
			params.HTTPListener.Close()
			params.HTTPSListener.Close()
			return err
		}
	}
	shutdownWaitGroup.Wait()
	log.Printf("All HTTP servers have shut down successfully")

	return nil
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
	gh := &gatedHandler{
		secret: params.Secret,
		inner:  params.Handler,
	}

	var opts []jsonrpc2.ConnOpt

	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	stream := jsonrpc2.NewBufferedStream(tcpConn, LFObjectCodec{})

	conn := jsonrpc2.NewConn(ctx, stream, gh, opts...)
	<-conn.DisconnectNotify()

	return nil
}

//

type gatedHandler struct {
	authenticated bool
	secret        string
	inner         jsonrpc2.Handler
}

var _ jsonrpc2.Handler = (*gatedHandler)(nil)

func (h *gatedHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if req.Method == "Meta.Authenticate" {
		err := func() error {
			var params MetaAuthenticateParams

			err := json.Unmarshal(*req.Params, &params)
			if err != nil {
				return errors.WithStack(err)
			}

			if params.Secret != h.secret {
				return errors.Errorf("Invalid secret")
			}
			return nil
		}()

		if err != nil {
			conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    jsonrpc2.CodeInvalidRequest,
				Message: fmt.Sprintf("%+v", err),
			})
		} else {
			result := &MetaAuthenticateResult{OK: true}
			h.authenticated = true
			conn.Reply(ctx, req.ID, result)
		}
	} else {
		if h.authenticated {
			go h.inner.Handle(ctx, conn, req)
		} else {
			conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    jsonrpc2.CodeInvalidRequest,
				Message: "Must call Meta.Authenticate with valid secret first",
			})
		}
	}
}

//

type LFObjectCodec struct{}

var separator = []byte("\n")

func (LFObjectCodec) WriteObject(stream io.Writer, obj interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	if _, err := stream.Write(data); err != nil {
		return err
	}
	if _, err := stream.Write(separator); err != nil {
		return err
	}
	return nil
}

func (LFObjectCodec) ReadObject(stream *bufio.Reader, v interface{}) error {
	var buf bytes.Buffer

scanLoop:
	for {
		b, err := stream.ReadByte()
		if err != nil {
			return err
		}

		switch b {
		case '\n':
			break scanLoop
		default:
			buf.WriteByte(b)
		}
	}

	return json.Unmarshal(buf.Bytes(), v)
}

type Conn interface {
	Notify(ctx context.Context, method string, params interface{}) error
	Call(ctx context.Context, method string, params interface{}, result interface{}) error
}

//

type JsonRPC2Conn struct {
	Conn *jsonrpc2.Conn
}

var _ Conn = (*JsonRPC2Conn)(nil)

func (jc *JsonRPC2Conn) Notify(ctx context.Context, method string, params interface{}) error {
	return jc.Conn.Notify(ctx, method, params)
}

func (jc *JsonRPC2Conn) Call(ctx context.Context, method string, params interface{}, result interface{}) error {
	return jc.Conn.Call(ctx, method, params, result)
}

func (jc *JsonRPC2Conn) Close() error {
	return jc.Conn.Close()
}
