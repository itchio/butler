package service

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"time"

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
	ctx *mansion.Context
}

func (h *handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
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
		case "Operation.Start":
			{
				startParams := &buse.OperationStartParams{}
				err := json.Unmarshal(*req.Params, startParams)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				comm.Logf("Starting operation in %s...", startParams.Params.StagingFolder)
				max := 5
				for i := 0; i < max; i++ {
					time.Sleep(100 * time.Millisecond)
					conn.Notify(ctx, "Operation.Progress", &buse.OperationProgressNotification{
						Progress: float64(i) / float64(max),
					})
				}

				return conn.Reply(ctx, req.ID, &buse.OperationResult{
					Success:      false,
					ErrorMessage: "stub!",
				})
			}
		}

		return nil
	}()

	if err != nil {
		comm.Warnf("error dealing with %s request: %s", req.Method, err.Error())
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
	err = s.Serve(context.Background(), lis, ha)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
