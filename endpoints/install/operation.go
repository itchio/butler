package install

import (
	"context"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/operate"
)

func OperationStart(rc *buse.RequestContext, params *buse.OperationStartParams) (*buse.OperationStartResult, error) {
	if params.ID == "" {
		return nil, errors.New("Missing ID")
	}

	parentCtx := rc.Ctx
	ctx, cancelFunc := context.WithCancel(parentCtx)

	rc.CancelFuncs.Add(params.ID, cancelFunc)
	defer rc.CancelFuncs.Remove(params.ID)

	err := operate.Start(ctx, rc.Conn, params)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &buse.OperationStartResult{}, nil
}

func OperationCancel(rc *buse.RequestContext, params *buse.OperationCancelParams) (*buse.OperationCancelResult, error) {
	rc.CancelFuncs.Call(params.ID)
	return &buse.OperationCancelResult{}, nil
}
