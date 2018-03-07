package install

import (
	"context"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/operate"
)

func InstallPerform(rc *buse.RequestContext, params *buse.InstallPerformParams) (*buse.InstallPerformResult, error) {
	if params.ID == "" {
		return nil, errors.New("Missing ID")
	}

	parentCtx := rc.Ctx
	ctx, cancelFunc := context.WithCancel(parentCtx)

	rc.CancelFuncs.Add(params.ID, cancelFunc)
	defer rc.CancelFuncs.Remove(params.ID)

	err := operate.InstallPerform(ctx, rc, params)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &buse.InstallPerformResult{}, nil
}
