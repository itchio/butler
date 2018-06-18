package install

import "github.com/itchio/butler/butlerd"

func InstallCancel(rc *butlerd.RequestContext, params butlerd.InstallCancelParams) (*butlerd.InstallCancelResult, error) {
	didCancel := rc.CancelFuncs.Call(params.ID)
	return &butlerd.InstallCancelResult{
		DidCancel: didCancel,
	}, nil
}
