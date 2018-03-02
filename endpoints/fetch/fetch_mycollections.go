package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/go-itchio"
	"github.com/jinzhu/gorm"
)

func FetchMyCollections(rc *buse.RequestContext, params *buse.FetchMyCollectionsParams) (*buse.FetchMyCollectionsResult, error) {
	consumer := rc.Consumer

	client, err := rc.SessionClient(params.SessionID)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	profile := &models.Profile{}
	err = db.Where("id = ?", params.SessionID).First(profile).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	sendDBCollections := func() error {
		var collections []*itchio.Collection
		err = db.Model(profile).Related(&collections, "Collections").Error
		if err != nil {
			return errors.Wrap(err, 0)
		}

		err = db.Preload("Games", func(db *gorm.DB) *gorm.DB {
			return db.Order(`"order" ASC`).Limit(8)
		}).Error
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if len(collections) > 0 {
			yn := &buse.FetchMyCollectionsYieldNotification{}
			yn.Offset = 0
			yn.Total = int64(len(collections))

			for _, c := range collections {
				cs := &buse.CollectionSummary{
					Collection: c,
				}

				for i, g := range c.Games {
					cs.Items = append(cs.Items, &buse.CollectionGame{
						Order: int64(i),
						Game:  g,
					})
				}

				yn.Items = append(yn.Items, cs)
			}

			err = messages.FetchMyCollectionsYield.Notify(rc, yn)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
		return nil
	}

	err = sendDBCollections()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	collRes, err := client.ListMyCollections()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	profile.Collections = collRes.Collections
	err = SaveRecursive(db, consumer, profile, []string{"Collections"})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = sendDBCollections()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.FetchMyCollectionsResult{}
	return res, nil
}
