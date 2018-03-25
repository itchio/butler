package downloads

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
)

func DownloadsPrioritize(rc *butlerd.RequestContext, params *butlerd.DownloadsPrioritizeParams) (*butlerd.DownloadsPrioritizeResult, error) {
	download := ValidateDownload(rc, params.DownloadID)

	download.Position = models.DownloadMinPosition(rc.DB()) - 1
	download.Save(rc.DB())

	res := &butlerd.DownloadsPrioritizeResult{}
	return res, nil
}
