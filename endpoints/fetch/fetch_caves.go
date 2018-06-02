package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
)

func FetchCaves(rc *butlerd.RequestContext, params *butlerd.FetchCavesParams) (*butlerd.FetchCavesResult, error) {
	var caves []*models.Cave
	var formattedCaves []*butlerd.Cave
	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustSelect(conn, &caves, builder.NewCond(), nil)
		models.PreloadCaves(conn, caves)
		for _, cave := range caves {
			formattedCaves = append(formattedCaves, FormatCave(conn, cave))
		}
	})

	res := &butlerd.FetchCavesResult{
		Caves: formattedCaves,
	}
	return res, nil
}
