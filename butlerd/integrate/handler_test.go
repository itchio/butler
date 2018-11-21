package integrate

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/wharf/state"
	"github.com/sourcegraph/jsonrpc2"
)

type handler struct {
	handlers             map[string]butlerd.RequestHandler
	notificationHandlers map[string]butlerd.NotificationHandler
	consumer             *state.Consumer
}

var _ jsonrpc2.Handler = (*handler)(nil)

func newHandler(consumer *state.Consumer) *handler {
	return &handler{
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
	logf := t.Logf
	if os.Getenv("LOUD_TESTS") == "1" {
		logf = func(msg string, args ...interface{}) {
			log.Printf(msg, args...)
		}
	}

	return connectEx(logf)
}

func connectEx(logf func(msg string, args ...interface{})) (*butlerd.RequestContext, *handler, context.CancelFunc) {
	consumer := &state.Consumer{
		OnMessage: func(lvl string, msg string) {
			logf("[%s] %s", lvl, msg)
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	h := newHandler(consumer)

	messages.Log.Register(h, func(rc *butlerd.RequestContext, params butlerd.LogNotification) {
		if consumer != nil && consumer.OnMessage != nil {
			consumer.OnMessage(string(params.Level), params.Message)
		}
	})

	tcpConn, err := net.DialTimeout("tcp", address, 2*time.Second)
	gmust(err)

	stream := jsonrpc2.NewBufferedStream(tcpConn, butlerd.LFObjectCodec{})

	jc := jsonrpc2.NewConn(ctx, stream, jsonrpc2.AsyncHandler(h))
	go func() {
		<-ctx.Done()
		jc.Close()
	}()

	rc := &butlerd.RequestContext{
		Conn:     &butlerd.JsonRPC2Conn{Conn: jc},
		Ctx:      ctx,
		Consumer: consumer,
	}

	_, err = messages.MetaAuthenticate.TestCall(rc, butlerd.MetaAuthenticateParams{
		Secret: secret,
	})
	gmust(err)

	return rc, h, cancel
}
