package install

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/pkg/errors"
)

func CavesSetPinned(rc *butlerd.RequestContext, params butlerd.CavesSetPinnedParams) (*butlerd.CavesSetPinnedResult, error) {
	var opErr error

	rc.WithConn(func(conn *sqlite.Conn) {
		cave := models.CaveByID(conn, params.CaveID)
		if cave == nil {
			opErr = errors.Errorf("cave (%s) not found", params.CaveID)
			return
		}
		cave.Pinned = params.Pinned
		cave.Save(conn)
	})

	if opErr != nil {
		return nil, opErr
	}

	return &butlerd.CavesSetPinnedResult{}, nil
}
