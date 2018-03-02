package fetch

import (
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/database"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
)

func FetchMyOwnedKeys(rc *buse.RequestContext, params *buse.FetchMyOwnedKeysParams) (*buse.FetchMyOwnedKeysResult, error) {
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

	sendDBKeys := func() error {
		// TODO: remove
		return nil

		var keys []*itchio.DownloadKey
		err := db.Model(profile).Related(&keys, "OwnedKeys").Error
		if err != nil {
			return errors.Wrap(err, 0)
		}

		yn := &buse.FetchMyOwnedKeysYieldNotification{
			Offset: 0,
			Total:  int64(len(keys)),
			Items:  keys,
		}
		err = messages.FetchMyOwnedKeysYield.Notify(rc, yn)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}

	err = sendDBKeys()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	ownedRes, err := client.ListMyOwnedKeys()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Infof("Saving %d keys", len(ownedRes.OwnedKeys))

	beforeDiff := time.Now()
	err = func() error {
		tx := db.Begin()
		success := false

		database.SetLogger(tx, consumer)

		defer func() {
			if success {
				tx.Commit()
			} else {
				tx.Rollback()
			}
		}()

		var err error

		err = diff(tx, consumer, ownedRes.OwnedKeys)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		var newGames []interface{}
		for _, k := range ownedRes.OwnedKeys {
			newGames = append(newGames, k.Game)
		}

		err = diff(tx, consumer, newGames)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		var newUsers []interface{}
		for _, k := range ownedRes.OwnedKeys {
			if k.Game != nil && k.Game.User != nil {
				newUsers = append(newUsers, k.Game.User)
			}
		}

		err = diff(tx, consumer, newUsers)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		// gameIDMap := make(map[int64]bool)
		// var gameIDs []int64
		// for _, k := range ownedRes.OwnedKeys {
		// 	id := k.Game.ID
		// 	if _, ok := gameIDMap[id]; !ok {
		// 		gameIDs = append(gameIDs, id)
		// 		gameIDMap[id] = true
		// 	}
		// }

		// var oldGames []*itchio.Game
		// err := tx.Where("id in (?)", gameIDs).Find(&oldGames).Error
		// if err != nil {
		// 	return errors.Wrap(err, 0)
		// }

		// consumer.Infof("Already have %d/%d games", len(oldGames), len(gameIDs))

		// gameMap := make(map[int64]*itchio.Game)
		// for _, g := range oldGames {
		// 	gameMap[g.ID] = g
		// }

		// numChanged := 0
		// for _, k := range ownedRes.OwnedKeys {
		// 	if g, ok := gameMap[k.Game.ID]; ok {
		// 		h := k.Game

		// 		if !RecordEqual(*g, *h) {
		// 			consumer.Infof("Game %s has changed:", g.Title)
		// 			numChanged++
		// 		}
		// 	}
		// }

		// consumer.Infof("%d/%d records have changed", numChanged, len(ownedRes.OwnedKeys))

		success = true
		return nil
	}()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Infof("Diff ran in %s", time.Since(beforeDiff))

	// {
	// 	var keys []interface{}
	// 	for _, k := range ownedRes.OwnedKeys {
	// 		k.OwnerID = profile.UserID
	// 		keys = append(keys, k)
	// 	}
	// 	tx := db.Begin()

	// 	beforeQueue := time.Now()
	// 	err := tx.Model(profile).Association("OwnedKeys").Clear().Error
	// 	if err != nil {
	// 		tx.Rollback()
	// 		return nil, errors.Wrap(err, 0)
	// 	}

	// 	err = tx.Model(profile).Association("OwnedKeys").Append(keys...).Error
	// 	if err != nil {
	// 		tx.Rollback()
	// 		return nil, errors.Wrap(err, 0)
	// 	}
	// 	consumer.Logf("Queuing took %s", time.Since(beforeQueue))

	// 	beforeCommit := time.Now()
	// 	err = tx.Commit().Error
	// 	if err != nil {
	// 		return nil, errors.Wrap(err, 0)
	// 	}
	// 	consumer.Logf("Commit took %s", time.Since(beforeCommit))
	// }

	err = sendDBKeys()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.FetchMyOwnedKeysResult{}
	return res, nil
}
