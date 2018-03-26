package fetch

import (
	"github.com/pkg/errors"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
)

func FetchCavesByInstallLocationID(rc *butlerd.RequestContext, params *butlerd.FetchCavesByInstallLocationIDParams) (*butlerd.FetchCavesByInstallLocationIDResult, error) {
	installLocation := models.InstallLocationByID(rc.DB(), params.InstallLocationID)
	if installLocation == nil {
		return nil, errors.Errorf("Install location not found (%s)", params.InstallLocationID)
	}

	caves := installLocation.GetCaves(rc.DB())
	models.PreloadCaves(rc.DB(), caves)

	var formattedCaves []*butlerd.Cave
	for _, c := range caves {
		formattedCaves = append(formattedCaves, FormatCave(rc.DB(), c))
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
