package install

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/operate"
)

func InstallQueue(rc *buse.RequestContext, params *buse.InstallQueueParams) (*buse.InstallQueueResult, error) {
	err := operate.InstallQueue(rc.Ctx, rc, params)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.InstallQueueResult{}
	return res, nil
}
