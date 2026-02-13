package install

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/pkg/errors"
)

func InstallPerform(rc *butlerd.RequestContext, params butlerd.InstallPerformParams) (*butlerd.InstallPerformResult, error) {
	if params.ID == "" {
		return nil, errors.New("Missing ID")
	}

	ctx, cleanup := rc.MakeCancelable(params.ID)
	defer cleanup()

	res, err := operate.InstallPerform(ctx, rc, params)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
}
