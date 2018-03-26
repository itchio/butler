package fetch

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/system"
	"github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

func FetchCommons(rc *butlerd.RequestContext, params *butlerd.FetchCommonsParams) (*butlerd.FetchCommonsResult, error) {
	var err error

	var caves []*butlerd.CaveSummary
	err = rc.DB().Model(&models.Cave{}).
		Select("id, game_id, last_touched_at, seconds_run, installed_size").
		Scan(&caves).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var downloadKeys []*butlerd.DownloadKeySummary
	err = rc.DB().Model(&itchio.DownloadKey{}).
		Select("id, game_id, created_at").
		Scan(&downloadKeys).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var installLocations []*models.InstallLocation
	err = rc.DB().Find(&installLocations).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var flocs []*butlerd.InstallLocationSummary
	for _, il := range installLocations {
		flocs = append(flocs, FormatInstallLocation(rc, il))
	}

	res := &butlerd.FetchCommonsResult{
		Caves:            caves,
		DownloadKeys:     downloadKeys,
		InstallLocations: flocs,
	}
	return res, nil
}

func FormatInstallLocation(rc *butlerd.RequestContext, il *models.InstallLocation) *butlerd.InstallLocationSummary {
	sum := &butlerd.InstallLocationSummary{
		ID:   il.ID,
		Path: il.Path,
		SizeInfo: &butlerd.InstallLocationSizeInfo{
			InstalledSize: -1,
			FreeSize:      -1,
			TotalSize:     -1,
		},
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
		rc.Consumer.Warnf("Could not compute installed size: %s", err.Error())
	} else {
		sum.SizeInfo.InstalledSize = row.InstalledSize
	}

	stats, err := system.StatFS(il.Path)
	if err != nil {
		rc.Consumer.Warnf("Could not statFS (%s): %s", il.Path, err.Error())
	} else {
		sum.SizeInfo.FreeSize = stats.FreeSize
		sum.SizeInfo.TotalSize = stats.TotalSize
	}

	return sum
}
