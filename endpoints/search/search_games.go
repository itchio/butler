package search

import (
	"fmt"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"xorm.io/builder"
)

func SearchGames(rc *butlerd.RequestContext, params butlerd.SearchGamesParams) (*butlerd.SearchGamesResult, error) {
	if params.Query == "" {
		// return empty games set
		return &butlerd.SearchGamesResult{
			Games: nil,
		}, nil
	}

	var games []*itchio.Game

	doLocalSearch := func() {
		games = nil
		q := fmt.Sprintf("%%%s%%", params.Query)
		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustSelect(conn, &games,
				builder.Like{"lower(title)", q},
				hades.Search{}.Limit(4),
			)
		})
	}

	//----------------------------------
	// perform local search
	//----------------------------------

	doLocalSearch()

	return &butlerd.SearchGamesResult{
		Games: games,
	}, nil
}
