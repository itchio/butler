package install

import "github.com/itchio/butler/buse"

func InstallCancel(rc *buse.RequestContext, params *buse.InstallCancelParams) (*buse.InstallCancelResult, error) {
	didCancel := rc.CancelFuncs.Call(params.ID)
	return &buse.InstallCancelResult{
		DidCancel: didCancel,
	}, nil
}
