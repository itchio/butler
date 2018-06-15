package fetch

import (
	"time"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func FetchCollection(rc *butlerd.RequestContext, params *butlerd.FetchCollectionParams) (*butlerd.FetchCollectionResult, error) {
	if params.CollectionID == 0 {
		return nil, errors.New("collectionId must be non-zero")
	}

	consumer := rc.Consumer
	ft := models.FetchTarget{
		Type: "collection",
		ID:   params.CollectionID,
		TTL:  10 * time.Minute,
	}

	fresh := false
	res := &butlerd.FetchCollectionResult{}

	if params.Fresh {
		consumer.Infof("Doing remote fetch (Fresh specified)")
		fresh = true
	} else if rc.WithConnBool(ft.IsStale) {
		consumer.Infof("Returning stale info")
		res.Stale = true
	}

	if fresh {
		fts := []models.FetchTarget{ft}

		_, client := rc.ProfileClient(params.ProfileID)

		consumer.Debugf("Querying API...")
		collRes, err := client.GetCollection(itchio.GetCollectionParams{
			CollectionID: params.CollectionID,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		collection := collRes.Collection
		collection.CollectionGames = nil

		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustSave(conn, collRes.Collection)
		})

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

			collection.CollectionGames = append(collection.CollectionGames, gamesRes.CollectionGames...)

			rc.WithConn(func(conn *sqlite.Conn) {
				models.MustSave(conn, collection,
					hades.Assoc("CollectionGames",
						hades.Assoc("Game"),
					),
				)
			})

			offset += numPageGames

			if offset >= collection.GamesCount {
				// already fetched all or more?!
				break
			}
		}

		for _, cg := range collection.CollectionGames {
			g := cg.Game
			fts = append(fts, models.FetchTarget{
				ID:   g.ID,
				Type: "game",
				TTL:  10 * time.Minute,
			})
		}

		rc.WithConn(func(conn *sqlite.Conn) {
			for _, ft := range fts {
				// TODO: avoid n+1
				ft.MarkFresh(conn)
			}
			models.MustSave(conn, collection, hades.AssocReplace("CollectionGames"))
		})
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		res.Collection = models.CollectionByID(conn, params.CollectionID)
	})

	if res.Collection == nil && !params.Fresh {
		freshParams := *params
		freshParams.Fresh = true
		return FetchCollection(rc, &freshParams)
	}

	return res, nil
}
