package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/launch"
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
	ctx              *mansion.Context
	operationHandles map[string]*operationHandle
}

type operationHandle struct {
	id         string
	cancelFunc context.CancelFunc
}

func (h *handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	comm.Warnf("Got request %s", req.Method)

	if req.Notif {
		log.Printf("Got a notif! method = %#v, params = %#v", req.Method, req.Params)
		return
	}

	err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				if rErr, ok := r.(error); ok {
					err = errors.Wrap(rErr, 0)
				} else {
					err = errors.New(r)
				}
			}
		}()

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
		case "CleanDownloads.Search":
			{
				params := &buse.CleanDownloadsSearchParams{}
				err := json.Unmarshal(*req.Params, params)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				consumer, err := operate.NewStateConsumer(&operate.NewStateConsumerParams{
					Conn: conn,
					Ctx:  ctx,
				})
				if err != nil {
					return errors.Wrap(err, 0)
				}

				res, err := operate.CleanDownloadsSearch(params, consumer)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				return conn.Reply(ctx, req.ID, res)
			}

		case "CleanDownloads.Apply":
			{
				params := &buse.CleanDownloadsApplyParams{}
				err := json.Unmarshal(*req.Params, params)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				consumer, err := operate.NewStateConsumer(&operate.NewStateConsumerParams{
					Conn: conn,
					Ctx:  ctx,
				})
				if err != nil {
					return errors.Wrap(err, 0)
				}

				res, err := operate.CleanDownloadsApply(params, consumer)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				return conn.Reply(ctx, req.ID, res)
			}
		case "Launch":
			{
				params := &buse.LaunchParams{}
				err := json.Unmarshal(*req.Params, params)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				err = launch.Do(ctx, conn, params)
				if err != nil {
					return errors.Wrap(err, 0)
				}
				return conn.Reply(ctx, req.ID, &buse.LaunchResult{})
			}
		case "Operation.Start":
			{
				params := &buse.OperationStartParams{}
				err := json.Unmarshal(*req.Params, params)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				if params.ID == "" {
					return errors.New("'id' parameter missing")
				}

				if _, ok := h.operationHandles[params.ID]; ok {
					return fmt.Errorf("an operation is already running with id '%s'", params.ID)
				}

				parentCtx := h.ctx.Context()
				ctx, cancelFunc := context.WithCancel(parentCtx)

				oh := &operationHandle{
					id:         params.ID,
					cancelFunc: cancelFunc,
				}
				h.operationHandles[oh.id] = oh

				err = operate.Start(ctx, h.ctx, conn, params)
				delete(h.operationHandles, oh.id)
				if err != nil {
					if errors.Is(err, operate.ErrCancelled) {
						conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
							Code:    buse.CodeOperationCancelled,
							Message: err.Error(),
						})
						return nil
					}

					if errors.Is(err, operate.ErrAborted) {
						conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
							Code:    buse.CodeOperationAborted,
							Message: err.Error(),
						})
						return nil
					}

					return err
				}

				return conn.Reply(ctx, req.ID, &buse.OperationResult{})
			}
		case "Operation.Cancel":
			{
				var creq buse.OperationCancelParams
				err := json.Unmarshal(*req.Params, &creq)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				if oh, ok := h.operationHandles[creq.ID]; ok {
					oh.cancelFunc()
					return conn.Reply(ctx, req.ID, &buse.OperationCancelResult{})
				}

				return fmt.Errorf("no such operation: %s", creq.ID)
			}
		case "Game.FindUploads":
			{
				res, err := operate.GameFindUploads(h.ctx, conn, req)
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

		msg := err.Error()
		if se, ok := err.(*errors.Error); ok {
			msg = se.ErrorStack()
		}

		// will get dropped if not handled, that's ok
		conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInternalError,
			Message: msg,
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
		ctx:              ctx,
		operationHandles: make(map[string]*operationHandle),
	}
	aha := jsonrpc2.AsyncHandler(ha)

	err = s.Serve(context.Background(), lis, aha)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
