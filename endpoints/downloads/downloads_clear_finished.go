package downloads

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/models"
)

func DownloadsClearFinished(rc *buse.RequestContext, params *buse.DownloadsClearFinishedParams) (*buse.DownloadsClearFinishedResult, error) {
	req := rc.DB().Delete(&models.Download{}, `"finished_at" is not null`)
	err := req.Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.DownloadsClearFinishedResult{
		RemovedCount: req.RowsAffected,
	}
	return res, nil
}
