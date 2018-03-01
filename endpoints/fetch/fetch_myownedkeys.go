package fetch

import (
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
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

	var keys []interface{}
	for _, k := range ownedRes.OwnedKeys {
		k.OwnerID = profile.UserID
		keys = append(keys, k)
	}

	consumer.Infof("Saving %d keys", len(keys))

	{
		tx := db.Begin()

		beforeQueue := time.Now()
		err := tx.Model(profile).Association("OwnedKeys").Replace(keys...).Error
		if err != nil {
			tx.Rollback()
			return nil, errors.Wrap(err, 0)
		}
		consumer.Logf("Queuing took %s", time.Since(beforeQueue))

		beforeCommit := time.Now()
		err = tx.Commit().Error
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		consumer.Logf("Commit took %s", time.Since(beforeCommit))
	}

	err = sendDBKeys()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.FetchMyOwnedKeysResult{}
	return res, nil
}
