package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/go-itchio"
)

func FetchMyCollections(rc *buse.RequestContext, params *buse.FetchMyCollectionsParams) (*buse.FetchMyCollectionsResult, error) {
	err := checkCredentials(params.Credentials)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var userCollections []*models.UserCollection
	err = db.Where("user_id = ?", params.Credentials.SessionID).Find(&userCollections).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if len(userCollections) > 0 {
		yn := &buse.FetchMyCollectionsYieldNotification{}
		yn.Offset = 0
		yn.Total = int64(len(userCollections))

		var collectionIDs []int64
		for _, uc := range userCollections {
			collectionIDs = append(collectionIDs, uc.CollectionID)
		}
		var collections []*itchio.Collection
		err = db.Where("where id in ?", collectionIDs).Find(collections).Error
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		for _, c := range collections {
			cs := &buse.CollectionSummary{
				Collection: c,
			}

			var games []*itchio.Game
			err = db.Table("collection_games").
				Where("collection_id = ?", c.ID).
				Order("collection_games.order desc").
				Limit(8).
				Joins("join games on collection_games.game_id = games.id").
				Scan(games).Error
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}

			for i, g := range games {
				cs.Items = append(cs.Items, &buse.CollectionGame{
					Order: int64(i),
					Game:  g,
				})
			}

			yn.Items = append(yn.Items, cs)
		}

		err = messages.FetchMyCollectionsYield.Notify(rc, yn)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	client, err := rc.Client(params.Credentials)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	collRes, err := client.ListMyCollections()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	{
		yg := &buse.FetchMyCollectionsYieldNotification{}
		yg.Offset = 0
		yg.Total = int64(len(collRes.Collections))

		for _, coll := range collRes.Collections {
			cs := &buse.CollectionSummary{
				Collection: coll,
			}

			for i, g := range coll.Games {
				cs.Items = append(cs.Items, &buse.CollectionGame{
					Order: int64(i),
					Game:  g,
				})
			}
			coll.Games = nil

			yg.Items = append(yg.Items, cs)
		}

		err = messages.FetchMyCollectionsYield.Notify(rc, yg)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	// TODO: persist

	res := &buse.FetchMyCollectionsResult{}
	return res, nil
}
