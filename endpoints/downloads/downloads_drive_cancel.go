package downloads

import "github.com/itchio/butler/butlerd"

func DownloadsDriveCancel(rc *butlerd.RequestContext, params *butlerd.DownloadsDriveCancelParams) (*butlerd.DownloadsDriveCancelResult, error) {
	didCancel := rc.CancelFuncs.Call(downloadsDriveCancelID)
	return &butlerd.DownloadsDriveCancelResult{
		DidCancel: didCancel,
	}, nil
}
