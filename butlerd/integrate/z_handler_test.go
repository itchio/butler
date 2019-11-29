package integrate

import (
	"fmt"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/itchio/headway/state"
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

func (h *handler) HandleNotification(conn jsonrpc2.Conn, notif jsonrpc2.Notification) {
	if nh, ok := h.notificationHandlers[notif.Method]; ok {
		nh(notif)
	}
	return
}

func (h *handler) HandleRequest(conn jsonrpc2.Conn, req jsonrpc2.Request) (interface{}, error) {
	rc := &butlerd.RequestContext{
		Ctx:      conn.Context(),
		Conn:     conn,
		Params:   req.Params,
		Consumer: h.consumer,
	}

	if rh, ok := h.handlers[req.Method]; ok {
		return rh(rc)
	}

	return nil, &jsonrpc2.Error{
		Code:    jsonrpc2.CodeMethodNotFound,
		Message: fmt.Sprintf("Method '%s' not found", req.Method),
	}
}

func (h *handler) Register(method string, rh butlerd.RequestHandler) {
	h.handlers[method] = rh
}

func (h *handler) RegisterNotification(method string, nh butlerd.NotificationHandler) {
	h.notificationHandlers[method] = nh
}
