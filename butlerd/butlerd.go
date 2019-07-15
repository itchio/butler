package butlerd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/itchio/butler/comm"
	"github.com/itchio/headway/state"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"
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
			err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    jsonrpc2.CodeInvalidRequest,
				Message: fmt.Sprintf("%+v", err),
			})
			if err != nil {
				comm.Warnf("Failed to reply: %#v", err)
			}
		} else {
			result := &MetaAuthenticateResult{OK: true}
			h.authenticated = true
			err := conn.Reply(ctx, req.ID, result)
			if err != nil {
				comm.Warnf("Failed to reply: %#v", err)
			}
		}
	} else {
		if h.authenticated {
			go h.inner.Handle(ctx, conn, req)
		} else {
			err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    jsonrpc2.CodeInvalidRequest,
				Message: "Must call Meta.Authenticate with valid secret first",
			})
			if err != nil {
				comm.Warnf("Failed to reply with error: %#v", err)
			}
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
