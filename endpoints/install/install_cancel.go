package install

import "github.com/itchio/butler/buse"

func InstallCancel(rc *buse.RequestContext, params *buse.InstallCancelParams) (*buse.InstallCancelResult, error) {
	rc.CancelFuncs.Call(params.ID)
	return &buse.InstallCancelResult{}, nil
}
