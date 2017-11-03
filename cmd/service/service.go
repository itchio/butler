package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/operate"
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
	ctx *mansion.Context
}

func (h *handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	comm.Warnf("got request %s", req.Method)

	if req.Notif {
		log.Printf("got a notif! method = %#v, params = %#v", req.Method, req.Params)
		return
	}

	err := func() error {
		switch req.Method {
		case "Version.Get":
			{
				return conn.Reply(ctx, req.ID, &buse.VersionGetResult{
					Version:       h.ctx.Version,
					VersionString: h.ctx.VersionString,
				})
			}
		case "Test.DoubleTwice":
			var ddreq buse.TestDoubleTwiceRequest
			err := json.Unmarshal(*req.Params, &ddreq)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			var dres buse.TestDoubleResult
			err = conn.Call(ctx, "Test.Double", &buse.TestDoubleRequest{Number: ddreq.Number}, &dres)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			return conn.Reply(ctx, req.ID, &buse.TestDoubleTwiceResult{
				Number: dres.Number * 2,
			})
		case "Operation.Start":
			{
				res, err := operate.Start(h.ctx, conn, req)
				if err != nil {
					return err
				}

				return conn.Reply(ctx, req.ID, res)
			}
		default:
			conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    jsonrpc2.CodeMethodNotFound,
				Message: fmt.Sprintf("no such method '%s'", req.Method),
			})
		}

		return nil
	}()

	if err != nil {
		comm.Warnf("error dealing with %s request: %s", req.Method, err.Error())

		// will get dropped if not handled, that's ok
		conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInternalError,
			Message: err.Error(),
		})
	}
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
		ctx: ctx,
	}
	aha := jsonrpc2.AsyncHandler(ha)

	err = s.Serve(context.Background(), lis, aha)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
