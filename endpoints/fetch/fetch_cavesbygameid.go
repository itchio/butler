package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/pkg/errors"
)

func FetchCavesByGameID(rc *butlerd.RequestContext, params *butlerd.FetchCavesByGameIDParams) (*butlerd.FetchCavesByGameIDResult, error) {
	if params.GameID == 0 {
		return nil, errors.New("gameId must be set")
	}

	var formattedCaves []*butlerd.Cave
	rc.WithConn(func(conn *sqlite.Conn) {
		caves := models.CavesByGameID(conn, params.GameID)
		models.PreloadCaves(conn, caves)
		for _, c := range caves {
			formattedCaves = append(formattedCaves, FormatCave(conn, c))
		}
	})

	res := &butlerd.FetchCavesByGameIDResult{
		Caves: formattedCaves,
	}
	return res, nil
}
