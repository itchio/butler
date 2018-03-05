package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/database/hades"
	itchio "github.com/itchio/go-itchio"
)

func FetchProfileOwnedKeys(rc *buse.RequestContext, params *buse.FetchProfileOwnedKeysParams) (*buse.FetchProfileOwnedKeysResult, error) {
	consumer := rc.Consumer

	profile, client, err := rc.ProfileClient(params.ProfileID)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	sendDBKeys := func() error {
		var keys []*itchio.DownloadKey
		err := db.Model(profile).Preload("Game").Related(&keys, "OwnedKeys").Error
		if err != nil {
			return errors.Wrap(err, 0)
		}

		yn := &buse.FetchProfileOwnedKeysYieldNotification{
			Offset: 0,
			Total:  int64(len(keys)),
			Items:  keys,
		}
		err = messages.FetchProfileOwnedKeysYield.Notify(rc, yn)
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

	c := hades.NewContext(db, consumer)

	profile.OwnedKeys = ownedRes.OwnedKeys
	err = c.Save(db, &hades.SaveParams{
		Record: profile,
		Assocs: []string{"OwnedKeys"},
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = sendDBKeys()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.FetchProfileOwnedKeysResult{}
	return res, nil
}
