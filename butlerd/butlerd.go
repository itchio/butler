package butlerd

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/sourcegraph/jsonrpc2"
)

type Server struct {
	secret string
}

func NewServer(secret string) *Server {
	return &Server{secret: secret}
}

type gatedHandler struct {
	h        jsonrpc2.Handler
	c        chan struct{}
	unlocked bool
}

func (gh *gatedHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if !gh.unlocked {
		select {
		case <-gh.c:
			gh.unlocked = true
		case <-ctx.Done():
			return
		}
	}
	gh.h.Handle(ctx, conn, req)
}

func (s *Server) Serve(ctx context.Context, lis net.Listener, h jsonrpc2.Handler, consumer *state.Consumer, opt ...jsonrpc2.ConnOpt) error {
	cancel := make(chan struct{})
	conns := make(chan net.Conn)
	disconnects := make(chan struct{})
	defer close(cancel)

	numClients := 0

	go func() {
		defer close(conns)

		for {
			conn, err := lis.Accept()
			if err != nil {
				consumer.Warnf("While accepting: %s", err.Error())
			}
			conns <- conn
		}
	}()

	for {
		select {
		case conn := <-conns:
			handshakeDone := make(chan struct{})
			gh := &gatedHandler{
				h: h,
				c: handshakeDone,
			}
			agh := jsonrpc2.AsyncHandler(gh)

			connCtx, cancelFunc := context.WithCancel(ctx)

			jc := jsonrpc2.NewConn(connCtx, jsonrpc2.NewBufferedStream(conn, LFObjectCodec{}), agh, opt...)
			numClients++
			consumer.Infof("butlerd: Accepted connection! (%d clients now)", numClients)
			go func() {
				<-jc.DisconnectNotify()
				cancelFunc()
				disconnects <- struct{}{}
			}()

			generateMessage := func() (string, error) {
				msg := ""
				for i := 0; i < 4; i++ {
					u, err := uuid.NewV4()
					if err != nil {
						return "", errors.Wrap(err, "generating UUID")
					}
					msg += u.String()
				}
				return msg, nil
			}

			go func() {
				die := func(msg string, args ...interface{}) {
					fmsg := fmt.Sprintf(msg, args...)
					consumer.Warnf("%s", fmsg)
					jc.Notify(ctx, "Log", LogNotification{
						Level:   "error",
						Message: fmsg,
					})
					jc.Close()
				}

				hres := &HandshakeResult{}
				message, err := generateMessage()
				if err != nil {
					die("butlerd: Message generation error: %s", err.Error())
					return
				}

				err = jc.Call(ctx, "Handshake", &HandshakeParams{
					Message: message,
				}, hres)
				if err != nil {
					die("butlerd: Handshake error: %s", err.Error())
					return
				}

				expectedSigBytes := sha256.Sum256([]byte(s.secret + message))
				expectedSig := fmt.Sprintf("%x", expectedSigBytes)
				if expectedSig != hres.Signature {
					die("butlerd: Handshake failed")
					return
				}

				close(handshakeDone)
			}()

			go func() {
				select {
				case <-handshakeDone:
					// good!
				case <-time.After(15 * time.Second):
					consumer.Warnf("butlerd: Handshake timed out!")
					jc.Close()
				}
			}()
		case <-disconnects:
			numClients--
			consumer.Infof("butlerd: Client disconnected! (%d clients left)", numClients)
			if numClients == 0 {
				consumer.Infof("butlerd: Last client left, shutting down")
				return nil
			}
		}
	}
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
