package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	itchio "github.com/itchio/go-itchio"
)

func FetchCollection(rc *buse.RequestContext, params *buse.FetchCollectionParams) (*buse.FetchCollectionResult, error) {
	consumer := rc.Consumer

	if params.CollectionID == 0 {
		return nil, errors.New("collectionId must be non-zero")
	}

	client, err := rc.SessionClient(params.SessionID)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	hadLocal := false
	collection := &itchio.Collection{}
	req := db.Where("id = ?", params.CollectionID).First(collection)
	if req.Error != nil {
		if !req.RecordNotFound() {
			return nil, errors.Wrap(req.Error, 0)
		}
	} else {
		hadLocal = true
		consumer.Infof("Yielding cached collection")
		err = messages.FetchCollectionYield.Notify(rc, &buse.FetchCollectionYieldNotification{Collection: collection})
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	consumer.Infof("Querying API...")
	collRes, err := client.GetCollection(&itchio.GetCollectionParams{
		CollectionID: params.CollectionID,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = messages.FetchCollectionYield.Notify(rc, &buse.FetchCollectionYieldNotification{Collection: collRes.Collection})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if hadLocal {
		// TODO: persist
		consumer.Infof("should persist collection: stub")
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

		ygn := &buse.FetchCollectionYieldGamesNotification{}
		ygn.Offset = offset
		ygn.Total = offset + numPageGames
		for i, game := range gamesRes.Games {
			cg := &buse.CollectionGame{
				Order: offset + int64(i),
				Game:  game,
			}
			ygn.Items = append(ygn.Items, cg)
		}
		err = messages.FetchCollectionYieldGames.Notify(rc, ygn)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		offset += numPageGames

		if offset >= collection.GamesCount {
			break
		}

		if hadLocal {
			// TODO: persist
			consumer.Infof("should persist collection games: stub")
		}
	}

	res := &buse.FetchCollectionResult{}
	return res, nil
}
