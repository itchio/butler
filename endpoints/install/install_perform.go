package install

import (
	"context"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
)

func InstallPerform(rc *butlerd.RequestContext, params *butlerd.InstallPerformParams) (*butlerd.InstallPerformResult, error) {
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

	return &butlerd.InstallPerformResult{}, nil
}
