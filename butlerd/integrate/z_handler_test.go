package integrate

import (
	"context"
	"fmt"

	"github.com/itchio/butler/butlerd"
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
			must(conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    jsonrpc2.CodeInternalError,
				Message: fmt.Sprintf("%+v", err),
			}))
			return
		}
		must(conn.Reply(ctx, req.ID, res))
		return
	}

	must(conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
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
