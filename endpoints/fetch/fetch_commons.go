package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/system"
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

	var installLocations []*models.InstallLocation
	err = rc.DB().Find(&installLocations).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var flocs []*buse.InstallLocationSummary
	for _, il := range installLocations {
		flocs = append(flocs, FormatInstallLocation(rc, il))
	}

	res := &buse.FetchCommonsResult{
		Caves:            caves,
		DownloadKeys:     downloadKeys,
		InstallLocations: flocs,
	}
	return res, nil
}

func FormatInstallLocation(rc *buse.RequestContext, il *models.InstallLocation) *buse.InstallLocationSummary {
	sum := &buse.InstallLocationSummary{
		ID:   il.ID,
		Path: il.Path,
	}

	var row struct {
		InstalledSize int64
	}
	err := rc.DB().Raw(`
		SELECT coalesce(sum(coalesce(installed_size, 0)), 0) AS installed_size
		FROM caves
		WHERE install_location_id = ?
	`, il.ID).Scan(&row).Error
	if err != nil {
		panic(err)
	}

	stats, err := system.StatFS(il.Path)
	if err != nil {
		panic(err)
	}

	sum.SizeInfo = &buse.InstallLocationSizeInfo{
		InstalledSize: row.InstalledSize,
		TotalSize:     stats.TotalSize,
		FreeSize:      stats.FreeSize,
	}
	return sum
}
