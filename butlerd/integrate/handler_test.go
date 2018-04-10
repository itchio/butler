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
	"github.com/itchio/wharf/state"
	"github.com/sourcegraph/jsonrpc2"
)

type handler struct {
	secret               string
	handlers             map[string]butlerd.RequestHandler
	notificationHandlers map[string]butlerd.NotificationHandler
	consumer             *state.Consumer
}

var _ jsonrpc2.Handler = (*handler)(nil)

func newHandler(secret string, consumer *state.Consumer) *handler {
	return &handler{
		secret:               secret,
		handlers:             make(map[string]butlerd.RequestHandler),
		notificationHandlers: make(map[string]butlerd.NotificationHandler),
		consumer:             consumer,
	}
}

func (h *handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	rc := &butlerd.RequestContext{
		Ctx:      ctx,
		Conn:     &butlerd.JsonRPC2Conn{Conn: conn},
		Params:   req.Params,
		Consumer: h.consumer,
	}

	if req.Notif {
		if nh, ok := h.notificationHandlers[req.Method]; ok {
			nh(rc)
		}
		return
	}

	if rh, ok := h.handlers[req.Method]; ok {
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
		Message: fmt.Sprintf("Method '%s' not found", req.Method),
	}))
}

func (h *handler) Register(method string, rh butlerd.RequestHandler) {
	h.handlers[method] = rh
}

func (h *handler) RegisterNotification(method string, nh butlerd.NotificationHandler) {
	h.notificationHandlers[method] = nh
}

func connect(t *testing.T) (*butlerd.RequestContext, *handler, context.CancelFunc) {
	return connectEx(t.Logf)
}

func connectEx(logf func(msg string, args ...interface{})) (*butlerd.RequestContext, *handler, context.CancelFunc) {
	conn, err := net.DialTimeout("tcp", address, time.Second)
	gmust(err)

	consumer := &state.Consumer{
		OnMessage: func(lvl string, msg string) {
			logf("[%s] %s", lvl, msg)
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	h := newHandler(secret, consumer)

	messages.Handshake.TestRegister(h, func(rc *butlerd.RequestContext, params *butlerd.HandshakeParams) (*butlerd.HandshakeResult, error) {
		return &butlerd.HandshakeResult{
			Signature: fmt.Sprintf("%x", sha256.Sum256([]byte(h.secret+params.Message))),
		}, nil
	})

	messages.Log.Register(h, func(rc *butlerd.RequestContext, params *butlerd.LogNotification) {
		if consumer != nil && consumer.OnMessage != nil {
			consumer.OnMessage(string(params.Level), params.Message)
		}
	})

	jc := jsonrpc2.NewConn(ctx, jsonrpc2.NewBufferedStream(conn, butlerd.LFObjectCodec{}), jsonrpc2.AsyncHandler(h))
	go func() {
		<-ctx.Done()
		<-time.After(1 * time.Second)
		jc.Close()
	}()

	return &butlerd.RequestContext{
		Conn:     &butlerd.JsonRPC2Conn{Conn: jc},
		Ctx:      ctx,
		Consumer: consumer,
	}, h, cancel
}
