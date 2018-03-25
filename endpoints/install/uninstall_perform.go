package install

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
)

func UninstallPerform(rc *butlerd.RequestContext, params *butlerd.UninstallPerformParams) (*butlerd.UninstallPerformResult, error) {
	err := operate.UninstallPerform(rc.Ctx, rc, params)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &butlerd.UninstallPerformResult{}
	return res, nil
}
