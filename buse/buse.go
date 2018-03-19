package buse

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/state"
	"github.com/sourcegraph/jsonrpc2"
)

type Server struct {
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Serve(ctx context.Context, lis net.Listener, h jsonrpc2.Handler, consumer *state.Consumer, opt ...jsonrpc2.ConnOpt) error {
	conn, err := lis.Accept()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	jc := jsonrpc2.NewConn(ctx, jsonrpc2.NewBufferedStream(conn, LFObjectCodec{}), h, opt...)
	consumer.Debugf("buse: Accepted connection!")
	<-jc.DisconnectNotify()
	consumer.Debugf("buse: Disconnected!")
	return nil
}

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
	var buf []byte

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
			buf = append(buf, b)
		}
	}

	return json.Unmarshal(buf, v)
}

type Conn interface {
	Notify(ctx context.Context, method string, params interface{}) error
	Call(ctx context.Context, method string, params interface{}, result interface{}) error
}

//

type jsonrpc2Conn struct {
	conn *jsonrpc2.Conn
}

var _ Conn = (*jsonrpc2Conn)(nil)

func (jc *jsonrpc2Conn) Notify(ctx context.Context, method string, params interface{}) error {
	return jc.conn.Notify(ctx, method, params)
}

func (jc *jsonrpc2Conn) Call(ctx context.Context, method string, params interface{}, result interface{}) error {
	return jc.conn.Call(ctx, method, params, result)
}

func (jc *jsonrpc2Conn) Close() error {
	return jc.conn.Close()
}
