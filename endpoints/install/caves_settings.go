package install

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/pkg/errors"
)

func CavesGetSettings(rc *butlerd.RequestContext, params butlerd.CavesGetSettingsParams) (*butlerd.CavesGetSettingsResult, error) {
	var res *butlerd.CavesGetSettingsResult
	var opErr error

	rc.WithConn(func(conn *sqlite.Conn) {
		cave := models.CaveByID(conn, params.CaveID)
		if cave == nil {
			opErr = errors.Errorf("cave (%s) not found", params.CaveID)
			return
		}

		var settings butlerd.CaveSettings
		err := models.UnmarshalJSONAllowEmpty(cave.Settings, &settings, "cave settings")
		if err != nil {
			opErr = errors.WithStack(err)
			return
		}

		res = &butlerd.CavesGetSettingsResult{
			Settings: settings,
		}
	})

	if opErr != nil {
		return nil, opErr
	}

	return res, nil
}

func CavesSetSettings(rc *butlerd.RequestContext, params butlerd.CavesSetSettingsParams) (*butlerd.CavesSetSettingsResult, error) {
	var opErr error

	rc.WithConn(func(conn *sqlite.Conn) {
		cave := models.CaveByID(conn, params.CaveID)
		if cave == nil {
			opErr = errors.Errorf("cave (%s) not found", params.CaveID)
			return
		}

		err := models.MarshalJSON(params.Settings, &cave.Settings, "cave settings")
		if err != nil {
			opErr = errors.WithStack(err)
			return
		}

		cave.Save(conn)
	})

	if opErr != nil {
		return nil, opErr
	}

	return &butlerd.CavesSetSettingsResult{}, nil
}
