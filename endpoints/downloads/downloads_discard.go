package downloads

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/models"
)

func DownloadsDiscard(rc *buse.RequestContext, params *buse.DownloadsDiscardParams) (*buse.DownloadsDiscardResult, error) {
	download := ValidateDownload(rc, params.DownloadID)

	// TODO: check whether it's dangerous to discard or not (if cave will be left morphing)
	err := rc.DB().Delete(download).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.DownloadsDiscardResult{}
	return res, nil
}

func ValidateDownload(rc *buse.RequestContext, downloadID string) *models.Download {
	if downloadID == "" {
		panic(errors.Errorf("downloadId must be set"))
	}
	download := models.DownloadByID(rc.DB(), downloadID)

	if download == nil {
		panic(errors.Errorf("Download not found (%s)", downloadID))
	}

	return download
}
