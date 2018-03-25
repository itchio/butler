package downloads

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
)

func DownloadsClearFinished(rc *butlerd.RequestContext, params *butlerd.DownloadsClearFinishedParams) (*butlerd.DownloadsClearFinishedResult, error) {
	req := rc.DB().Model(&models.Download{}).Where(`finished_at IS NOT NULL`).Update("discarded", true)
	err := req.Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &butlerd.DownloadsClearFinishedResult{
		RemovedCount: req.RowsAffected,
	}
	return res, nil
}
