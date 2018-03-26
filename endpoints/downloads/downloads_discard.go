package downloads

import (
	"github.com/pkg/errors"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
)

func DownloadsDiscard(rc *butlerd.RequestContext, params *butlerd.DownloadsDiscardParams) (*butlerd.DownloadsDiscardResult, error) {
	consumer := rc.Consumer
	download := ValidateDownload(rc, params.DownloadID)

	if download.Discarded {
		consumer.Warnf("Download already discarded")
	} else {
		consumer.Statf("Discarded download for %s", operate.GameToString(download.Game))

		// TODO: check whether it's dangerous to discard or not (if cave will be left morphing)
		download.Discarded = true
		download.Save(rc.DB())
	}

	res := &butlerd.DownloadsDiscardResult{}
	return res, nil
}

func ValidateDownload(rc *butlerd.RequestContext, downloadID string) *models.Download {
	if downloadID == "" {
		panic(errors.Errorf("downloadId must be set"))
	}
	download := models.DownloadByID(rc.DB(), downloadID)

	if download == nil {
		panic(errors.Errorf("Download not found (%s)", downloadID))
	}

	download.Preload(rc.DB())

	return download
}
