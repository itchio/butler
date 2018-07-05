package butlerd

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/itchio/wharf/state"
	"github.com/sourcegraph/jsonrpc2"
)

type Server struct {
	secret string
}

func NewServer(secret string) *Server {
	return &Server{secret: secret}
}

type ServeParams struct {
	HTTPListener net.Listener

	HTTPSListener net.Listener
	TLSState      *TLSState

	Handler  jsonrpc2.Handler
	Consumer *state.Consumer
}

func (s *Server) Serve(ctx context.Context, params ServeParams, opt ...jsonrpc2.ConnOpt) error {
	hh := &httpHandler{
		jrh:    params.Handler,
		secret: s.secret,
	}
	lh := handlers.LoggingHandler(os.Stderr, hh)

	errors := make(chan error)
	go func() {
		tlsListener := tls.NewListener(params.HTTPSListener, params.TLSState.Config)
		srv := &http.Server{Handler: lh}
		srv.TLSConfig = params.TLSState.Config
		errors <- srv.Serve(tlsListener)
	}()

	go func() {
		srv := &http.Server{Handler: lh}
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
}

func (s *Server) ServeTCP(ctx context.Context, params ServeTCPParams) error {
	tcpConn, err := params.Listener.Accept()
	if err != nil {
		return err
	}

	stream := jsonrpc2.NewBufferedStream(tcpConn, LFObjectCodec{})

	logger := log.New(os.Stderr, "[rpc]", log.LstdFlags)
	conn := jsonrpc2.NewConn(ctx, stream, jsonrpc2.AsyncHandler(params.Handler), jsonrpc2.LogMessages(logger))
	<-conn.DisconnectNotify()
	return nil
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
