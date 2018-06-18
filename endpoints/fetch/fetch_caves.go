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
	res := &butlerd.FetchCavesResult{}

	rc.WithConn(func(conn *sqlite.Conn) {
		var cond = builder.NewCond()

		var search hades.Search
		switch params.SortBy {
		case "title":
			ordering := pager.Ordering("ASC", params.Reverse)
			search = search.OrderBy("games.title "+ordering).Join("games", "games.id = caves.game_id")
		case "lastTouched", "":
			ordering := pager.Ordering("DESC", params.Reverse)
			search = search.OrderBy("caves.last_touched_at " + ordering)
		}

		var items []*models.Cave
		pg := pager.New(params)
		res.NextCursor = pg.Fetch(conn, &items, cond, search)
		models.PreloadCaves(conn, items)
		for _, cave := range items {
			res.Caves = append(res.Caves, FormatCave(conn, cave))
		}
	})
	return res, nil
}
