package fetch

import (
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/models"
	"github.com/jinzhu/gorm"
)

func FetchCave(rc *buse.RequestContext, params *buse.FetchCaveParams) (*buse.FetchCaveResult, error) {
	cave := models.CaveByID(rc.DB(), params.CaveID)
	cave.Preload(rc.DB())

	res := &buse.FetchCaveResult{
		Cave: FormatCave(rc.DB(), cave),
	}
	return res, nil
}

func FormatCave(db *gorm.DB, cave *models.Cave) *buse.Cave {
	if cave == nil {
		return nil
	}

	return &buse.Cave{
		ID: cave.ID,

		Game:   cave.Game,
		Upload: cave.Upload,
		Build:  cave.Build,

		InstallInfo: &buse.CaveInstallInfo{
			InstallFolder:   cave.GetInstallFolder(db),
			InstalledSize:   cave.InstalledSize,
			InstallLocation: cave.InstallLocationID,
		},

		Stats: &buse.CaveStats{
			InstalledAt:   cave.InstalledAt,
			LastTouchedAt: cave.LastTouchedAt,
			SecondsRun:    cave.SecondsRun,
		},
	}
}
