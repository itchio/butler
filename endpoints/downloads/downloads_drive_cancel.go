package downloads

import "github.com/itchio/butler/buse"

func DownloadsDriveCancel(rc *buse.RequestContext, params *buse.DownloadsDriveCancelParams) (*buse.DownloadsDriveCancelResult, error) {
	rc.CancelFuncs.Call(downloadsDriveCancelID)
	return &buse.DownloadsDriveCancelResult{}, nil
}
