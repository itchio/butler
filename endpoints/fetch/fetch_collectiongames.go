package fetch

import (
	"time"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/pager"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func FetchCollectionGames(rc *butlerd.RequestContext, params *butlerd.FetchCollectionGamesParams) (*butlerd.FetchCollectionGamesResult, error) {
	if params.CollectionID == 0 {
		return nil, errors.New("collectionId must be non-zero")
	}

	consumer := rc.Consumer
	ft := models.FetchTarget{
		Type: "collection_games",
		ID:   params.CollectionID,
		TTL:  30 * time.Minute,
	}

	fresh := false
	res := &butlerd.FetchCollectionGamesResult{}

	if params.Fresh {
		consumer.Infof("Doing remote fetch (Fresh specified)")
		fresh = true
	} else if rc.WithConnBool(ft.MustIsStale) {
		consumer.Infof("Returning stale info")
		res.Stale = true
	}

	if fresh {
		fts := []models.FetchTarget{ft}

		_, client := rc.ProfileClient(params.ProfileID)

		consumer.Debugf("Querying API...")
		var fakeColl = &itchio.Collection{
			ID: params.CollectionID,
		}
		var collectionGames []*itchio.CollectionGame

		var offset int64
		for page := int64(1); ; page++ {
			consumer.Infof("Fetching page %d", page)

			gamesRes, err := client.GetCollectionGames(itchio.GetCollectionGamesParams{
				CollectionID: params.CollectionID,
				Page:         page,
			})
			if err != nil {
				return nil, errors.WithStack(err)
			}
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
						hades.Assoc("Game"),
					),
				)
			})

			offset += numPageGames
		}

		for _, cg := range collectionGames {
			g := cg.Game
			fts = append(fts, models.FetchTarget{
				ID:   g.ID,
				Type: "game",
				TTL:  10 * time.Minute,
			})
		}

		rc.WithConn(func(conn *sqlite.Conn) {
			fakeColl.CollectionGames = collectionGames
			models.MustSave(conn, fakeColl,
				hades.OmitRoot(),
				hades.AssocReplace("CollectionGames"),
			)
			models.MustMarkAllFresh(conn, fts)
		})
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		var cond builder.Cond = builder.Eq{"collection_id": params.CollectionID}
		search := hades.Search{}.OrderBy("position ASC")

		var items []*itchio.CollectionGame
		pg := pager.New(params)
		res.NextCursor = pg.Fetch(conn, &items, cond, search)
		models.MustPreload(conn, items, hades.Assoc("Game"))
		res.Items = items
	})
	return res, nil
}
