package buse

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/sourcegraph/jsonrpc2"
)

type Server struct {
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Serve(ctx context.Context, lis net.Listener, h jsonrpc2.Handler, opt ...jsonrpc2.ConnOpt) error {
	conn, err := lis.Accept()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	jc := jsonrpc2.NewConn(ctx, jsonrpc2.NewBufferedStream(conn, LFObjectCodec{}), h, opt...)
	comm.Debugf("buse: Accepted connection!")
	<-jc.DisconnectNotify()
	comm.Debugf("buse: Disconected!")
	return nil
}

type LFObjectCodec struct {
}

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
