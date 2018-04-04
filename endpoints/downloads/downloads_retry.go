package downloads

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
)

func DownloadsRetry(rc *butlerd.RequestContext, params *butlerd.DownloadsRetryParams) (*butlerd.DownloadsRetryResult, error) {
	consumer := rc.Consumer

	download := ValidateDownload(rc, params.DownloadID)

	if download.Error == nil {
		consumer.Warnf("No error, can't retry download")
	} else {
		download.Error = nil
		download.ErrorCode = nil
		download.ErrorMessage = nil
		download.FinishedAt = nil
		download.Save(rc.DB())

		consumer.Statf("Queued a retry for download for %s", operate.GameToString(download.Game))
	}

	res := &butlerd.DownloadsRetryResult{}
	return res, nil
}
