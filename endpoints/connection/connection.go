package connection

import (
	"context"
	"net"

	"github.com/go-errors/errors"

	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/mansion"
	"github.com/sourcegraph/jsonrpc2"
)

func Register(router *buse.Router) {
	messages.ConnectionNew.Register(router, ConnectionNew)
}

// TODO: dedup code with command/service

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

func ConnectionNew(rc *buse.RequestContext, params *buse.ConnectionNewParams) (*buse.ConnectionNewResult, error) {
	listenSpec := "127.0.0.1:"

	lis, err := net.Listen("tcp", listenSpec)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	go func() {
		s := buse.NewServer()

		ha := &handler{
			ctx:    rc.MansionContext,
			router: rc.Router,
		}
		aha := jsonrpc2.AsyncHandler(ha)

		// TODO: timeout?
		err := s.Serve(context.Background(), lis, aha, rc.Consumer)
		if err != nil {
			rc.Consumer.Warnf("While handling Connection.New: %s", err.Error())
		}
	}()

	res := &buse.ConnectionNewResult{
		Address: lis.Addr().String(),
	}
	return res, nil
}
