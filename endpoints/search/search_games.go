package search

import (
	"fmt"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func SearchGames(rc *butlerd.RequestContext, params butlerd.SearchGamesParams) (*butlerd.SearchGamesResult, error) {
	if params.Query == "" {
		// return empty games set
		err := messages.SearchGamesYield.Notify(rc, butlerd.SearchGamesYieldNotification{
			Games: nil,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		return &butlerd.SearchGamesResult{}, nil
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
	// return results from local DB
	//----------------------------------

	doLocalSearch()
	err := messages.SearchGamesYield.Notify(rc, butlerd.SearchGamesYieldNotification{
		Games: games,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	//----------------------------------
	// perform API request
	//----------------------------------

	_, client := rc.ProfileClient(params.ProfileID)
	searchRes, err := client.SearchGames(itchio.SearchGamesParams{
		Query: params.Query,
		Page:  1,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	//----------------------------------
	// save remote results which were already in local cache
	//----------------------------------

	localMap := make(map[int64]bool)
	for _, g := range games {
		localMap[g.ID] = true
	}

	var updatedGames []*itchio.Game
	for _, g := range searchRes.Games {
		if localMap[g.ID] {
			updatedGames = append(updatedGames, g)
		}
	}
	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustSave(conn, updatedGames)
	})

	//----------------------------------
	// send local + remote results
	//----------------------------------

	doLocalSearch()
	for _, g := range searchRes.Games {
		if len(games) > 15 {
			break
		}

		if _, ok := localMap[g.ID]; !ok {
			games = append(games, g)
		}
	}

	err = messages.SearchGamesYield.Notify(rc, butlerd.SearchGamesYieldNotification{
		Games: games,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := &butlerd.SearchGamesResult{}
	return res, nil
}
