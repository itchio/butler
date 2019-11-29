package loopbackconn

import (
	"context"
	"fmt"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/jsonrpc2"

	"github.com/itchio/headway/state"
)

//

type NotificationHandler func(conn jsonrpc2.Conn, method string, params interface{}) error
type CallHandler func(conn jsonrpc2.Conn, method string, params interface{}, result interface{}) error

var NoopNotificationHandler NotificationHandler = func(conn jsonrpc2.Conn, method string, params interface{}) error {
	return nil
}

type LoopbackConn interface {
	jsonrpc2.Conn

	OnNotification(method string, handler NotificationHandler)
	OnCall(method string, handler CallHandler)
}

type loopbackConn struct {
	ctx                  context.Context
	consumer             *state.Consumer
	notificationHandlers map[string]NotificationHandler
	callHandlers         map[string]CallHandler
}

func New(ctx context.Context, consumer *state.Consumer) LoopbackConn {
	lc := &loopbackConn{
		ctx:                  ctx,
		consumer:             consumer,
		notificationHandlers: make(map[string]NotificationHandler),
		callHandlers:         make(map[string]CallHandler),
	}

	lc.OnNotification("Log", func(conn jsonrpc2.Conn, method string, params interface{}) error {
		log := params.(*butlerd.LogNotification)
		lc.consumer.OnMessage(string(log.Level), log.Message)
		return nil
	})

	return lc
}

var _ LoopbackConn = (*loopbackConn)(nil)

func (lc *loopbackConn) OnNotification(method string, handler NotificationHandler) {
	lc.notificationHandlers[method] = handler
}

func (lc *loopbackConn) Notify(method string, params interface{}) error {
	if h, ok := lc.notificationHandlers[method]; ok {
		return h(lc, method, params)
	}
	return nil
}

func (lc *loopbackConn) OnCall(method string, handler CallHandler) {
	lc.callHandlers[method] = handler
}

func (lc *loopbackConn) Call(method string, params interface{}, result interface{}) error {
	if h, ok := lc.callHandlers[method]; ok {
		return h(lc, method, params, result)
	}
	return fmt.Errorf("No handler registered for method (%s)", method)
}

func (lc *loopbackConn) Context() context.Context {
	return lc.ctx
}

func (lc *loopbackConn) Close() {
	// muffin.
}
