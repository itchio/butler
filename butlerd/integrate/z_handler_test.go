package integrate

import (
	"fmt"
	"sync"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/itchio/headway/state"
)

type handler struct {
	handlers      map[string]butlerd.RequestHandler
	handlersMutex sync.Mutex

	notificationHandlers      map[string]butlerd.NotificationHandler
	notificationHandlersMutex sync.Mutex

	consumer *state.Consumer
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
	h.notificationHandlersMutex.Lock()
	nh, ok := h.notificationHandlers[notif.Method]
	h.notificationHandlersMutex.Unlock()

	if ok {
		nh(notif)
	}
}

func (h *handler) HandleRequest(conn jsonrpc2.Conn, req jsonrpc2.Request) (interface{}, error) {
	rc := &butlerd.RequestContext{
		Ctx:      conn.Context(),
		Conn:     conn,
		Params:   req.Params,
		Consumer: h.consumer,
	}

	h.handlersMutex.Lock()
	rh, ok := h.handlers[req.Method]
	h.handlersMutex.Unlock()
	if ok {
		return rh(rc)
	}

	return nil, &jsonrpc2.Error{
		Code:    jsonrpc2.CodeMethodNotFound,
		Message: fmt.Sprintf("Method '%s' not found", req.Method),
	}
}

func (h *handler) Register(method string, rh butlerd.RequestHandler) {
	h.handlersMutex.Lock()
	h.handlers[method] = rh
	h.handlersMutex.Unlock()
}

func (h *handler) RegisterNotification(method string, nh butlerd.NotificationHandler) {
	h.notificationHandlersMutex.Lock()
	h.notificationHandlers[method] = nh
	h.notificationHandlersMutex.Unlock()
}
