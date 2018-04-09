package integrate

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/sourcegraph/jsonrpc2"
)

type handler struct {
	secret   string
	handlers map[string]butlerd.RequestHandler
}

var _ jsonrpc2.Handler = (*handler)(nil)

func newHandler(secret string) *handler {
	return &handler{
		secret:   secret,
		handlers: make(map[string]butlerd.RequestHandler),
	}
}

func (h *handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if rh, ok := h.handlers[req.Method]; ok {
		rc := &butlerd.RequestContext{
			Ctx:    ctx,
			Conn:   &butlerd.JsonRPC2Conn{Conn: conn},
			Params: req.Params,
		}
		res, err := rh(rc)
		if err != nil {
			gmust(conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    jsonrpc2.CodeInternalError,
				Message: fmt.Sprintf("%+v", err),
			}))
			return
		}
		gmust(conn.Reply(ctx, req.ID, res))
		return
	}

	gmust(conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
		Code:    jsonrpc2.CodeMethodNotFound,
		Message: "Method not found",
	}))
}

func (h *handler) Register(method string, rh butlerd.RequestHandler) {
	h.handlers[method] = rh
}

func connect(t *testing.T) *butlerd.RequestContext {
	conn, err := net.DialTimeout("tcp", address, time.Second)
	gmust(err)

	ctx := context.Background()
	h := newHandler(secret)

	messages.Handshake.TestRegister(h, func(rc *butlerd.RequestContext, params *butlerd.HandshakeParams) (*butlerd.HandshakeResult, error) {
		return &butlerd.HandshakeResult{
			Signature: fmt.Sprintf("%x", sha256.Sum256([]byte(h.secret+params.Message))),
		}, nil
	})

	ah := jsonrpc2.AsyncHandler(h)

	jc := jsonrpc2.NewConn(ctx, jsonrpc2.NewBufferedStream(conn, butlerd.LFObjectCodec{}), ah)
	return &butlerd.RequestContext{
		Conn: &butlerd.JsonRPC2Conn{Conn: jc},
		Ctx:  ctx,
	}
}
