package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/go-itchio"
)

func FetchCommons(rc *buse.RequestContext, params *buse.FetchCommonsParams) (*buse.FetchCommonsResult, error) {
	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var caves []*buse.CaveSummary
	err = db.Model(&models.Cave{}).
		Select("id, game_id, last_touched_at, seconds_run, installed_size").
		Scan(&caves).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var downloadKeys []*buse.DownloadKeySummary
	err = db.Model(&itchio.DownloadKey{}).
		Select("id, game_id, created_at").
		Scan(&downloadKeys).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var installLocations []*buse.InstallLocationSummary
	err = db.Raw(`
		SELECT sum(coalesce(installed_size, 0)) as size, install_location
		FROM caves
		GROUP BY install_location
	`).Scan(&installLocations).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.FetchCommonsResult{
		Caves:            caves,
		DownloadKeys:     downloadKeys,
		InstallLocations: installLocations,
	}
	return res, nil
}
