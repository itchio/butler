package service

import (
	"context"
	"log"
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
	ctx.Must(Do(ctx))
}

type handler struct {
	ctx              *mansion.Context
	operationHandles map[string]*operationHandle
	router           *buse.Router
}

type operationHandle struct {
	id         string
	cancelFunc context.CancelFunc
}

func (h *handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if req.Notif {
		log.Printf("Got a notif! method = %#v, params = %#v", req.Method, req.Params)
		return
	}

	h.router.Dispatch(ctx, conn, req)
}

func Do(ctx *mansion.Context) error {
	port := "127.0.0.1:"

	lis, err := net.Listen("tcp", port)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Result(map[string]interface{}{
		"type":    "server-listening",
		"address": lis.Addr().String(),
	})

	s := buse.NewServer()

	ha := &handler{
		ctx:              ctx,
		operationHandles: make(map[string]*operationHandle),
		router:           getRouter(ctx),
	}
	aha := jsonrpc2.AsyncHandler(ha)

	err = s.Serve(context.Background(), lis, aha)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
