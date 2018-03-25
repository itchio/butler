package fetch

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/jinzhu/gorm"
)

func FetchCave(rc *butlerd.RequestContext, params *butlerd.FetchCaveParams) (*butlerd.FetchCaveResult, error) {
	cave := models.CaveByID(rc.DB(), params.CaveID)
	cave.Preload(rc.DB())

	res := &butlerd.FetchCaveResult{
		Cave: FormatCave(rc.DB(), cave),
	}
	return res, nil
}

func FormatCave(db *gorm.DB, cave *models.Cave) *butlerd.Cave {
	if cave == nil {
		return nil
	}

	return &butlerd.Cave{
		ID: cave.ID,

		Game:   cave.Game,
		Upload: cave.Upload,
		Build:  cave.Build,

		InstallInfo: &butlerd.CaveInstallInfo{
			InstallFolder:   cave.GetInstallFolder(db),
			InstalledSize:   cave.InstalledSize,
			InstallLocation: cave.InstallLocationID,
		},

		Stats: &butlerd.CaveStats{
			InstalledAt:   cave.InstalledAt,
			LastTouchedAt: cave.LastTouchedAt,
			SecondsRun:    cave.SecondsRun,
		},
	}
}
