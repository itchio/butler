package downloads

import (
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/operate"
)

func DownloadsRetry(rc *buse.RequestContext, params *buse.DownloadsRetryParams) (*buse.DownloadsRetryResult, error) {
	consumer := rc.Consumer

	download := ValidateDownload(rc, params.DownloadID)

	if download.Error == nil {
		consumer.Warnf("No error, can't retry download")
	} else {
		download.Error = nil
		download.FinishedAt = nil
		download.Save(rc.DB())

		consumer.Statf("Queued a retry for download for %s", operate.GameToString(download.Game))
	}

	res := &buse.DownloadsRetryResult{}
	return res, nil
}
