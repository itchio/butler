package downloads

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	"github.com/pkg/errors"
)

func DownloadsDiscard(rc *butlerd.RequestContext, params butlerd.DownloadsDiscardParams) (*butlerd.DownloadsDiscardResult, error) {
	consumer := rc.Consumer
	rc.WithConn(func(conn *sqlite.Conn) {
		download := ValidateDownload(conn, params.DownloadID)
		if download.Discarded {
			consumer.Warnf("Download already discarded")
		} else {
			consumer.Statf("Discarded download for %s", operate.GameToString(download.Game))

			// TODO: check whether it's dangerous to discard or not (if cave will be left morphing)
			download.Discarded = true
			download.Save(conn)
		}
	})

	res := &butlerd.DownloadsDiscardResult{}
	return res, nil
}

func ValidateDownload(conn *sqlite.Conn, downloadID string) *models.Download {
	if downloadID == "" {
		panic(errors.Errorf("downloadId must be set"))
	}
	download := models.DownloadByID(conn, downloadID)
	if download == nil {
		panic(errors.Errorf("Download not found (%s)", downloadID))
	}
	download.Preload(conn)

	return download
}
