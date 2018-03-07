package install

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/operate"
)

func UninstallPerform(rc *buse.RequestContext, params *buse.UninstallPerformParams) (*buse.UninstallPerformResult, error) {
	err := operate.UninstallPerform(rc.Ctx, rc, params)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.UninstallPerformResult{}
	return res, nil
}
