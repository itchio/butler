package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/models"
)

func FetchCavesByInstallLocationID(rc *buse.RequestContext, params *buse.FetchCavesByInstallLocationIDParams) (*buse.FetchCavesByInstallLocationIDResult, error) {
	installLocation := models.InstallLocationByID(rc.DB(), params.InstallLocationID)
	if installLocation == nil {
		return nil, errors.Errorf("Install location not found (%s)", params.InstallLocationID)
	}

	caves := installLocation.GetCaves()
	models.PreloadCaves(rc.DB(), caves)

	var formattedCaves []*buse.Cave
	for _, c := range caves {
		formattedCaves = append(formattedCaves, formatCave(rc.DB(), c))
	}

	var totalSize int64
	for _, cave := range caves {
		totalSize += cave.InstalledSize
	}

	res := &buse.FetchCavesByInstallLocationIDResult{
		InstallLocationPath: installLocation.Path,
		InstallLocationSize: installLocationSize,
		Caves:               formattedCaves,
	}
	return res, nil
}
