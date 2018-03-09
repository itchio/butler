package downloads

import (
	"github.com/itchio/butler/buse"
)

func DownloadsRetry(rc *buse.RequestContext, params *buse.DownloadsRetryParams) (*buse.DownloadsRetryResult, error) {
	download := ValidateDownload(rc, params.DownloadID)

	download.Error = nil
	download.Save(rc.DB())

	res := &buse.DownloadsRetryResult{}
	return res, nil
}
