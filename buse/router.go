package buse

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/state"
	"github.com/sourcegraph/jsonrpc2"
)

type RequestHandler func(rc *RequestContext) (interface{}, error)

type Router struct {
	Handlers       map[string]RequestHandler
	MansionContext *mansion.Context
	CancelFuncs    *CancelFuncs
}

func NewRouter(mansionContext *mansion.Context) *Router {
	return &Router{
		Handlers:       make(map[string]RequestHandler),
		MansionContext: mansionContext,
		CancelFuncs: &CancelFuncs{
			Funcs: make(map[string]context.CancelFunc),
		},
	}
}

func (r *Router) Register(method string, rh RequestHandler) {
	if _, ok := r.Handlers[method]; ok {
		panic(fmt.Sprintf("Can't register handler twice for %s", method))
	}
	r.Handlers[method] = rh
}

func (r Router) Dispatch(ctx context.Context, origConn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	method := req.Method
	var res interface{}

	conn := &jsonrpc2Conn{origConn}
	consumer, cErr := NewStateConsumer(&NewStateConsumerParams{
		Ctx:  ctx,
		Conn: conn,
	})
	if cErr != nil {
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

		if h, ok := r.Handlers[method]; ok {
			rc := &RequestContext{
				Ctx:            ctx,
				Harness:        NewProductionHarness(),
				Consumer:       consumer,
				Params:         req.Params,
				Conn:           conn,
				MansionContext: r.MansionContext,
				CancelFuncs:    r.CancelFuncs,
			}
			res, err = h(rc)
		} else {
			err = StandardRpcError(jsonrpc2.CodeMethodNotFound)
		}
		return
	}()

	if err == nil {
		err = origConn.Reply(ctx, req.ID, res)
		if err != nil {
			consumer.Errorf("Error while replying: %s", err.Error())
		}
		return
	}

	if ee, ok := asBuseError(err); ok {
		origConn.ReplyWithError(ctx, req.ID, ee.AsJsonRpc2())
		return
	}

	var errStack *json.RawMessage
	if se, ok := err.(*errors.Error); ok {
		input := map[string]interface{}{
			"stack": se.ErrorStack(),
		}
		es, err := json.Marshal(input)
		if err == nil {
			rm := json.RawMessage(es)
			errStack = &rm
		}
	}
	origConn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
		Code:    jsonrpc2.CodeInternalError,
		Message: err.Error(),
		Data:    errStack,
	})
}

type RequestContext struct {
	Ctx            context.Context
	Harness        Harness
	Consumer       *state.Consumer
	Params         *json.RawMessage
	Conn           Conn
	MansionContext *mansion.Context
	CancelFuncs    *CancelFuncs
}

type WithParamsFunc func() (interface{}, error)

func (rc *RequestContext) WithParams(params interface{}, cb WithParamsFunc) (interface{}, error) {
	err := json.Unmarshal(*rc.Params, params)
	if err != nil {
		return nil, &RpcError{
			Code:    jsonrpc2.CodeParseError,
			Message: err.Error(),
		}
	}

	return cb()
}

func (rc *RequestContext) Call(method string, params interface{}, res interface{}) error {
	return rc.Conn.Call(rc.Ctx, method, params, res)
}

func (rc *RequestContext) Notify(method string, params interface{}) error {
	return rc.Conn.Notify(rc.Ctx, method, params)
}

type CancelFuncs struct {
	Funcs map[string]context.CancelFunc
}

func (cf *CancelFuncs) Add(id string, f context.CancelFunc) {
	cf.Funcs[id] = f
}

func (cf *CancelFuncs) Remove(id string) {
	delete(cf.Funcs, id)
}

func (cf *CancelFuncs) Call(id string) bool {
	if f, ok := cf.Funcs[id]; ok {
		f()
		delete(cf.Funcs, id)
		return true
	}

	return false
}