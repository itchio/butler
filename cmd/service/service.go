package service

import (
	"context"
	"net"

	"github.com/itchio/butler/buse"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
)

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("service", "Start up the butler service").Hidden()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, ctx.Context(), func(addr string) {
		comm.Result(map[string]interface{}{
			"type":    "server-listening",
			"address": addr,
		})
	}))
}

type handler struct {
	ctx    *mansion.Context
	router *buse.Router
}

func (h *handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if req.Notif {
		return
	}

	h.router.Dispatch(ctx, conn, req)
}

type OnListenFunc func(addr string)

func Do(mansionContext *mansion.Context, ctx context.Context, onListen OnListenFunc) error {
	listenSpec := "127.0.0.1:"

	lis, err := net.Listen("tcp", listenSpec)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	onListen(lis.Addr().String())
	s := buse.NewServer()

	ha := &handler{
		ctx:    mansionContext,
		router: getRouter(mansionContext),
	}
	aha := jsonrpc2.AsyncHandler(ha)

	err = s.Serve(ctx, lis, aha, comm.NewStateConsumer())
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
