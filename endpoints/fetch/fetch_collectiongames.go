package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/lazyfetch"
	"github.com/itchio/butler/endpoints/fetch/pager"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
	"xorm.io/builder"
)

func LazyFetchCollectionGames(rc *butlerd.RequestContext, params lazyfetch.ProfiledLazyFetchParams, res lazyfetch.LazyFetchResponse, collectionID int64) {
	ft := models.FetchTargetForCollectionGames(collectionID)
	lazyfetch.Do(rc, ft, params, res, func(targets lazyfetch.Targets) {
		_, client := rc.ProfileClient(params.GetProfileID())
		maxGamesToFetch := maxCollectionGamesLimit

		var fakeColl = &itchio.Collection{
			ID: collectionID,
		}
		var collectionGames []*itchio.CollectionGame

		for page := int64(1); ; page++ {
			if int64(len(collectionGames)) >= maxGamesToFetch {
				rc.Consumer.Warnf("Reached collection fetch cap (%d items), stopping early", maxGamesToFetch)
				break
			}

			rc.Consumer.Infof("Fetching page %d (of unknown)", page)

			gamesRes, err := client.GetCollectionGames(rc.Ctx, itchio.GetCollectionGamesParams{
				CollectionID: collectionID,
				Page:         page,
			})
			models.Must(err)
			numPageGames := int64(len(gamesRes.CollectionGames))

			if numPageGames == 0 {
				break
			}

			pageGames := gamesRes.CollectionGames
			remaining := maxGamesToFetch - int64(len(collectionGames))
			if int64(len(pageGames)) > remaining {
				pageGames = pageGames[:remaining]
			}

			collectionGames = append(collectionGames, pageGames...)

			rc.WithConn(func(conn *sqlite.Conn) {
				// Save only this page incrementally to avoid reprocessing all previous pages.
				fakeColl.CollectionGames = pageGames
				models.MustSave(conn, fakeColl,
					hades.OmitRoot(),
					hades.Assoc("CollectionGames",
						hades.Assoc("Game",
							hades.Assoc("Sale"),
						),
					),
				)
			})

			for _, cg := range pageGames {
				g := cg.Game
				targets.Add(models.FetchTargetForGame(g.ID))
			}

			if len(pageGames) < len(gamesRes.CollectionGames) {
				rc.Consumer.Warnf("Reached collection fetch cap (%d items), stopping early", maxGamesToFetch)
				break
			}
		}

		rc.WithConn(func(conn *sqlite.Conn) {
			fakeColl.CollectionGames = collectionGames
			models.MustSave(conn, fakeColl,
				hades.OmitRoot(),
				hades.AssocReplace("CollectionGames"),
			)
		})
	})
}

func FetchCollectionGames(rc *butlerd.RequestContext, params butlerd.FetchCollectionGamesParams) (*butlerd.FetchCollectionGamesResult, error) {
	if params.CollectionID == 0 {
		return nil, errors.New("collectionId must be non-zero")
	}
	res := &butlerd.FetchCollectionGamesResult{}
	LazyFetchCollectionGames(rc, params, res, params.CollectionID)

	rc.WithConn(func(conn *sqlite.Conn) {
		var cond builder.Cond = builder.Eq{"collection_id": params.CollectionID}
		joinGames := false
		search := hades.Search{}

		switch params.SortBy {
		case "default", "":
			search = search.OrderBy("position " + pager.Ordering("DESC", params.Reverse))
		case "title":
			search = search.OrderBy("lower(games.title) " + pager.Ordering("ASC", params.Reverse))
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
			search = search.InnerJoin("games", "games.id = collection_games.game_id")
		}

		var items []*itchio.CollectionGame
		pg := pager.New(params)
		res.NextCursor = pg.Fetch(conn, &items, cond, search)
		models.MustPreload(conn, items, hades.Assoc("Game"))
		res.Items = items
	})
	return res, nil
}
