package fetch

import (
	"strconv"
	"time"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
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

	limit := params.Limit
	if limit == 0 {
		limit = 5
	}
	consumer.Infof("Using limit of %d", limit)

	doRemoteFetch := false

	if params.IgnoreCache {
		consumer.Infof("Doing remote fetch (IgnoreCache specified)")
		doRemoteFetch = true
	} else if rc.WithConnBool(ft.IsStale) {
		consumer.Infof("Doing remote fetch (Is stale)")
		doRemoteFetch = true
	}

	if doRemoteFetch {
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

			for _, cg := range gamesRes.CollectionGames {
				collection.CollectionGames = append(collection.CollectionGames, cg)
			}

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

	res := &butlerd.FetchCollectionResult{}

	rc.WithConn(func(conn *sqlite.Conn) {
		collection := models.CollectionByID(conn, params.CollectionID)
		if collection == nil {
			return
		}
		res.Collection = collection

		var cond builder.Cond = builder.Eq{"collection_id": collection.ID}
		if params.Cursor != "" {
			cond = builder.And(cond, builder.Gte{"position": params.Cursor})
		}

		var cgs []*itchio.CollectionGame
		models.MustSelect(conn, &cgs, cond, hades.Search().OrderBy("position ASC").Limit(limit))
		models.MustPreload(conn, cgs, hades.Assoc("Game"))

		models.CollectionExt(collection).PreloadCollectionGames(conn)

		for i, cg := range cgs {
			if i == len(cgs)-1 {
				res.NextCursor = strconv.FormatInt(cg.Position, 10)
			} else {
				res.CollectionGames = append(res.CollectionGames, cg)
			}
		}
	})
	return res, nil
}
