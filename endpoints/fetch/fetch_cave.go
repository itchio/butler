package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
)

func FetchCave(rc *butlerd.RequestContext, params butlerd.FetchCaveParams) (*butlerd.FetchCaveResult, error) {
	var res *butlerd.FetchCaveResult
	rc.WithConn(func(conn *sqlite.Conn) {
		cave := models.CaveByID(conn, params.CaveID)
		cave.Preload(conn)
		res = &butlerd.FetchCaveResult{
			Cave: FormatCave(conn, cave),
		}
	})
	return res, nil
}

func FormatCave(conn *sqlite.Conn, cave *models.Cave) *butlerd.Cave {
	if cave == nil {
		return nil
	}

	return &butlerd.Cave{
		ID: cave.ID,

		Game:   cave.Game,
		Upload: cave.Upload,
		Build:  cave.Build,

		InstallInfo: &butlerd.CaveInstallInfo{
			InstallFolder:   cave.GetInstallFolder(conn),
			InstalledSize:   cave.InstalledSize,
			InstallLocation: cave.InstallLocationID,
			Pinned:          cave.Pinned,
		},

		Stats: &butlerd.CaveStats{
			InstalledAt:   cave.InstalledAt,
			LastTouchedAt: cave.LastTouchedAt,
			SecondsRun:    cave.SecondsRun,
		},
	}
}
