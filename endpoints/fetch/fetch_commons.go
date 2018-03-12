package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/go-itchio"
)

func FetchCommons(rc *buse.RequestContext, params *buse.FetchCommonsParams) (*buse.FetchCommonsResult, error) {
	var err error

	var caves []*buse.CaveSummary
	err = rc.DB().Model(&models.Cave{}).
		Select("id, game_id, last_touched_at, seconds_run, installed_size").
		Scan(&caves).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var downloadKeys []*buse.DownloadKeySummary
	err = rc.DB().Model(&itchio.DownloadKey{}).
		Select("id, game_id, created_at").
		Scan(&downloadKeys).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var rows []struct {
		InstalledSize int64
		ID            string
	}
	err = rc.DB().Raw(`
		SELECT
			sum(coalesce(installed_size, 0)) AS installed_size,
			install_location_id AS id
		FROM caves
		GROUP BY install_location_id
	`).Scan(&rows).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var installLocations []*buse.InstallLocationSummary
	for _, row := range rows {
		installLocations = append(installLocations, &buse.InstallLocationSummary{
			ID: row.ID,
			// TODO: what about path?
			SizeInfo: &buse.InstallLocationSizeInfo{
				InstalledSize: row.InstalledSize,
				// TODO: fill in
				TotalSize: -1,
				FreeSize:  -1,
			},
		})
	}

	res := &buse.FetchCommonsResult{
		Caves:            caves,
		DownloadKeys:     downloadKeys,
		InstallLocations: installLocations,
	}
	return res, nil
}
