package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/pager"
	"github.com/itchio/hades"
)

func FetchCaves(rc *butlerd.RequestContext, params butlerd.FetchCavesParams) (*butlerd.FetchCavesResult, error) {
	res := &butlerd.FetchCaveResult{}

	var formattedCaves []*butlerd.Cave
	rc.WithConn(func(conn *sqlite.Conn) {
		var cond = builder.NewCond()
		search := hades.Search{}.OrderBy("last_touched_at DESC")

		var items []*models.Cave
		pg := pager.New(params)
		res.NextCursor = pg.Fetch(conn, items, cond, search)
		models.PreloadCaves(conn, caves)
		for _, cave := range caves {
			res.Caves = append(res.Caves, FormatCave(conn, cave))
		}
	})
	return res, nil
}
