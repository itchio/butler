package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/models"
)

func FetchCave(rc *buse.RequestContext, params *buse.FetchCaveParams) (*buse.FetchCaveResult, error) {
	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	cave, err := models.CaveByID(db, params.CaveID)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if cave != nil {
		err = PreloadCaves(db, rc.Consumer, cave)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	res := &buse.FetchCaveResult{
		Cave: formatCave(cave),
	}
	return res, nil
}

func formatCave(cave *models.Cave) *buse.Cave {
	if cave == nil {
		return nil
	}

	return &buse.Cave{
		ID: cave.ID,

		Game:   cave.Game,
		Upload: cave.Upload,
		Build:  cave.Build,

		InstallInfo: &buse.CaveInstallInfo{
			AbsoluteInstallFolder: "<stub>",
			InstalledSize:         cave.InstalledSize,
			InstallLocation:       cave.InstallLocation,
		},

		Stats: &buse.CaveStats{
			InstalledAt:   cave.InstalledAt,
			LastTouchedAt: cave.LastTouchedAt,
			SecondsRun:    cave.SecondsRun,
		},
	}
}
