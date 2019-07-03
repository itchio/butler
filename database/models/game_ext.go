package models

import (
	"time"

	"crawshaw.io/sqlite"
	"xorm.io/builder"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
)

// Game is defined in `go-itchio`, but helper functions are here

func GameByID(conn *sqlite.Conn, id int64) *itchio.Game {
	var g itchio.Game
	if MustSelectOne(conn, &g, builder.Eq{"id": id}) {
		return &g
	}
	return nil
}

func MustPreloadGameSales(conn *sqlite.Conn, g *itchio.Game) {
	games := []*itchio.Game{g}
	MustPreloadGamesSales(conn, games)
}

func MustPreloadGamesSales(conn *sqlite.Conn, games []*itchio.Game) {
	// TODO: paginate if games is too large?

	var gameIDs []interface{}
	gamesByID := make(map[int64]*itchio.Game)
	for _, g := range games {
		if g != nil {
			gameIDs = append(gameIDs, g.ID)
			gamesByID[g.ID] = g
		}
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)

	var sales []*itchio.Sale

	conds := builder.And(
		builder.In("sales.game_id", gameIDs...),
		builder.Gt{"sales.end_date": now},
		builder.Lt{"sales.start_date": now},
	)
	search := hades.Search{}.GroupBy("sales.game_id").OrderBy("sales.rate DESC")
	MustSelect(conn, &sales, conds, search)

	for _, s := range sales {
		if g := gamesByID[s.GameID]; g != nil {
			g.Sale = s
		}
	}
}
