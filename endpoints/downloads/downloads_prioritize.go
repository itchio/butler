package downloads

import (
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/models"
)

func DownloadsPrioritize(rc *buse.RequestContext, params *buse.DownloadsPrioritizeParams) (*buse.DownloadsPrioritizeResult, error) {
	download := ValidateDownload(rc, params.DownloadID)

	download.Position = models.DownloadMinPosition(rc.DB()) - 1
	download.Save(rc.DB())

	res := &buse.DownloadsPrioritizeResult{}
	return res, nil
}
