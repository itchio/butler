package downloads

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
)

func DownloadsRetry(rc *butlerd.RequestContext, params *butlerd.DownloadsRetryParams) (*butlerd.DownloadsRetryResult, error) {
	consumer := rc.Consumer

	var download *models.Download
	rc.WithConn(func(conn *sqlite.Conn) {
		download = ValidateDownload(conn, params.DownloadID)
		if download.Error == nil {
			consumer.Warnf("No error, can't retry download")
		} else {
			download.Error = nil
			download.ErrorCode = nil
			download.ErrorMessage = nil
			download.FinishedAt = nil
			download.Save(conn)

			consumer.Statf("Queued a retry for download for %s", operate.GameToString(download.Game))
		}
	})

	res := &butlerd.DownloadsRetryResult{}
	return res, nil
}
