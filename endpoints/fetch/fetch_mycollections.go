package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/go-itchio"
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
		var profileCollections []*models.ProfileCollection
		err = db.Model(profile).
			Preload("Collection").
			Order(`"position" DESC`).
			Related(&profileCollections, "ProfileCollections").
			Error
		if err != nil {
			return errors.Wrap(err, 0)
		}

		var collectionIDs []int64
		collectionsByIDs := make(map[int64]*itchio.Collection)
		for _, pc := range profileCollections {
			c := pc.Collection
			collectionIDs = append(collectionIDs, c.ID)
			collectionsByIDs[c.ID] = c
		}

		// TODO: figure out if we can't just pass `itchio.CollectionGame` here
		var cgs []struct {
			itchio.CollectionGame
			itchio.Game
		}
		err := db.Raw(`
			SELECT collection_games.*, games.*
			FROM collections
			JOIN collection_games ON collection_games.collection_id = collections.id
			JOIN games ON games.id = collection_games.game_id
			WHERE collections.id IN (?)
			AND collection_games.game_id IN (
				SELECT game_id
				FROM collection_games
				WHERE collection_games.collection_id = collections.id
				ORDER BY "position" ASC
				LIMIT 8
			)
		`, collectionIDs).Scan(&cgs).Error
		if err != nil {
			return errors.Wrap(err, 0)
		}

		for _, cg := range cgs {
			c := collectionsByIDs[cg.CollectionGame.CollectionID]
			cg.CollectionGame.Game = &cg.Game
			c.CollectionGames = append(c.CollectionGames, &cg.CollectionGame)
		}

		if len(profileCollections) > 0 {
			yn := &buse.FetchMyCollectionsYieldNotification{}
			yn.Offset = 0
			yn.Total = int64(len(profileCollections))

			for _, pc := range profileCollections {
				yn.Items = append(yn.Items, pc.Collection)
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

	profile.ProfileCollections = nil
	for i, c := range collRes.Collections {
		for j, g := range c.Games {
			c.CollectionGames = append(c.CollectionGames, &itchio.CollectionGame{
				Position: int64(j),
				Game:     g,
			})
		}
		c.Games = nil

		profile.ProfileCollections = append(profile.ProfileCollections, &models.ProfileCollection{
			// Other fields are set when saving the association
			Collection: c,
			Position:   int64(i),
		})
	}

	err = SaveRecursive(db, consumer, &SaveParams{
		Record: profile,
		Assocs: []string{"ProfileCollections"},

		PartialJoins: []string{"CollectionGames"},
	})
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
