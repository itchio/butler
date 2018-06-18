package fetch

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/pkg/errors"
)

func FetchCavesByInstallLocationID(rc *butlerd.RequestContext, params butlerd.FetchCavesByInstallLocationIDParams) (*butlerd.FetchCavesByInstallLocationIDResult, error) {
	conn := rc.DBPool.Get(rc.Ctx.Done())
	defer rc.DBPool.Put(conn)

	installLocation := models.InstallLocationByID(conn, params.InstallLocationID)
	if installLocation == nil {
		return nil, errors.Errorf("Install location not found (%s)", params.InstallLocationID)
	}

	caves := installLocation.GetCaves(conn)
	models.PreloadCaves(conn, caves)

	var formattedCaves []*butlerd.Cave
	for _, c := range caves {
		formattedCaves = append(formattedCaves, FormatCave(conn, c))
	}

	var totalSize int64
	for _, cave := range caves {
		totalSize += cave.InstalledSize
	}

	res := &butlerd.FetchCavesByInstallLocationIDResult{
		InstallLocationPath: installLocation.Path,
		InstallLocationSize: totalSize,
		Caves:               formattedCaves,
	}
	return res, nil
}
