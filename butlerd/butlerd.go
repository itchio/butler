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

	Log bool
}

func (s *Server) ServeHTTP(ctx context.Context, params ServeHTTPParams) error {
	hh := &httpHandler{
		jrh:    params.Handler,
		secret: s.secret,
	}

	var chosenHandler http.Handler = hh
	if params.Log {
		chosenHandler = handlers.LoggingHandler(os.Stderr, chosenHandler)
	}

	errors := make(chan error)
	go func() {
		tlsListener := tls.NewListener(params.HTTPSListener, params.TLSState.Config)
		srv := &http.Server{Handler: chosenHandler}
		srv.TLSConfig = params.TLSState.Config
		errors <- srv.Serve(tlsListener)
	}()

	go func() {
		srv := &http.Server{Handler: chosenHandler}
		errors <- srv.Serve(params.HTTPListener)
	}()

	for i := 0; i < 2; i++ {
		err := <-errors
		if err != nil {
			params.HTTPListener.Close()
			params.HTTPSListener.Close()
			return err
		}
	}
	return nil
}

type ServeTCPParams struct {
	Handler  jsonrpc2.Handler
	Consumer *state.Consumer
	Listener net.Listener
	Secret   string
	Log      bool
}

func (s *Server) ServeTCP(ctx context.Context, params ServeTCPParams) error {
	tcpConn, err := params.Listener.Accept()
	if err != nil {
		return err
	}

	logger := log.New(os.Stderr, "[rpc]", log.LstdFlags)
	gh := &gatedHandler{
		secret: params.Secret,
		inner:  params.Handler,
	}

	var opts []jsonrpc2.ConnOpt
	if params.Log {
		opts = append(opts, jsonrpc2.LogMessages(logger))
	}

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
