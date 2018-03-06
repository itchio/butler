package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/database/hades"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
)

func FetchCollection(rc *buse.RequestContext, params *buse.FetchCollectionParams) (*buse.FetchCollectionResult, error) {
	consumer := rc.Consumer

	if params.CollectionID == 0 {
		return nil, errors.New("collectionId must be non-zero")
	}

	_, client, err := rc.ProfileClient(params.ProfileID)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	sendDBCollection := func() error {
		collection, err := models.CollectionByID(db, params.CollectionID)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if collection == nil {
			return nil
		}

		err = hades.NewContext(db, consumer).Preload(db, &hades.PreloadParams{
			Record: collection,
			Fields: []hades.PreloadField{
				hades.PreloadField{Name: "CollectionGames", OrderBy: `"position" ASC`},
				hades.PreloadField{Name: "CollectionGames.Game"},
			},
		})
		if err != nil {
			return errors.Wrap(err, 0)
		}

		err = messages.FetchCollectionYield.Notify(rc, &buse.FetchCollectionYieldNotification{Collection: collection})
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}

	err = sendDBCollection()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Infof("Querying API...")
	collRes, err := client.GetCollection(&itchio.GetCollectionParams{
		CollectionID: params.CollectionID,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	collection := collRes.Collection
	collection.Games = nil
	collection.CollectionGames = nil

	c := hades.NewContext(db, consumer)

	err = c.Save(db, &hades.SaveParams{
		Record: collRes.Collection,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	// after collection metadata update
	err = sendDBCollection()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var offset int64
	for page := int64(1); ; page++ {
		consumer.Infof("Fetching page %d", page)

		gamesRes, err := client.GetCollectionGames(&itchio.GetCollectionGamesParams{
			CollectionID: params.CollectionID,
			Page:         page,
		})
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		numPageGames := int64(len(gamesRes.Games))

		if numPageGames == 0 {
			break
		}

		for i, game := range gamesRes.Games {
			collection.CollectionGames = append(collection.CollectionGames, &itchio.CollectionGame{
				Position: offset + int64(i),
				Game:     game,
			})
		}

		err = c.Save(db, &hades.SaveParams{
			Record: collection,
			Assocs: []string{"CollectionGames"},

			PartialJoins: []string{"CollectionGames"},
		})
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		offset += numPageGames

		if offset >= collection.GamesCount {
			// already fetched all or more?!
			break
		}

		if numPageGames < gamesRes.PerPage {
			// that probably means there's no more pages
			break
		}

		// after each page of games fetched
		err = sendDBCollection()
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	err = c.Save(db, &hades.SaveParams{
		Record: collection,
		Assocs: []string{"CollectionGames"},
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	// after all pages are fetched
	err = sendDBCollection()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.FetchCollectionResult{}
	return res, nil
}
