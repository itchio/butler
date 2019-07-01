package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/lazyfetch"
	"github.com/itchio/butler/endpoints/fetch/pager"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func FetchCollectionGames(rc *butlerd.RequestContext, params butlerd.FetchCollectionGamesParams) (*butlerd.FetchCollectionGamesResult, error) {
	if params.CollectionID == 0 {
		return nil, errors.New("collectionId must be non-zero")
	}

	ft := models.FetchTargetForCollectionGames(params.CollectionID)
	res := &butlerd.FetchCollectionGamesResult{}

	lazyfetch.Do(rc, ft, params, res, func(targets lazyfetch.Targets) {
		_, client := rc.ProfileClient(params.ProfileID)

		var fakeColl = &itchio.Collection{
			ID: params.CollectionID,
		}
		var collectionGames []*itchio.CollectionGame

		var offset int64
		for page := int64(1); ; page++ {
			rc.Consumer.Infof("Fetching page %d (of unknown)", page)

			gamesRes, err := client.GetCollectionGames(rc.Ctx, itchio.GetCollectionGamesParams{
				CollectionID: params.CollectionID,
				Page:         page,
			})
			models.Must(err)
			numPageGames := int64(len(gamesRes.CollectionGames))

			if numPageGames == 0 {
				break
			}

			collectionGames = append(collectionGames, gamesRes.CollectionGames...)

			rc.WithConn(func(conn *sqlite.Conn) {
				fakeColl.CollectionGames = collectionGames
				models.MustSave(conn, fakeColl,
					hades.OmitRoot(),
					hades.Assoc("CollectionGames",
						hades.Assoc("Game",
							hades.Assoc("Sale"),
						),
					),
				)
			})

			offset += numPageGames
		}

		for _, cg := range collectionGames {
			g := cg.Game
			targets.Add(models.FetchTargetForGame(g.ID))
		}

		rc.WithConn(func(conn *sqlite.Conn) {
			fakeColl.CollectionGames = collectionGames
			models.MustSave(conn, fakeColl,
				hades.OmitRoot(),
				hades.AssocReplace("CollectionGames"),
			)
		})
	})

	rc.WithConn(func(conn *sqlite.Conn) {
		var cond builder.Cond = builder.Eq{"collection_id": params.CollectionID}
		joinGames := false
		search := hades.Search{}

		switch params.SortBy {
		case "default", "":
			search = search.OrderBy("position " + pager.Ordering("DESC", params.Reverse))
		case "title":
			search = search.OrderBy("games.title " + pager.Ordering("ASC", params.Reverse))
			joinGames = true
		}

		if params.Filters.Installed {
			cond = builder.And(cond, builder.Expr("exists (select 1 from caves where caves.game_id = collection_games.game_id)"))
		}

		if params.Filters.Classification != "" {
			cond = builder.And(cond, builder.Eq{"games.classification": params.Filters.Classification})
			joinGames = true
		}

		if params.Search != "" {
			cond = builder.And(cond, builder.Like{"games.title", params.Search})
			joinGames = true
		}

		if joinGames {
			search = search.Join("games", "games.id = collection_games.game_id")
		}

		var items []*itchio.CollectionGame
		pg := pager.New(params)
		res.NextCursor = pg.Fetch(conn, &items, cond, search)
		models.MustPreload(conn, items, hades.Assoc("Game"))
		res.Items = items
	})
	return res, nil
}
