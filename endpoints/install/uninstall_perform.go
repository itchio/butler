package install

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/pkg/errors"
)

func UninstallPerform(rc *butlerd.RequestContext, params *butlerd.UninstallPerformParams) (*butlerd.UninstallPerformResult, error) {
	err := operate.UninstallPerform(rc.Ctx, rc, params)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := &butlerd.UninstallPerformResult{}
	return res, nil
}
