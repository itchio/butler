package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/database/hades"
)

func FetchProfileOwnedKeys(rc *butlerd.RequestContext, params *butlerd.FetchProfileOwnedKeysParams) (*butlerd.FetchProfileOwnedKeysResult, error) {
	profile, client := rc.ProfileClient(params.ProfileID)

	c := HadesContext(rc)

	sendDBKeys := func() error {
		err := c.Preload(rc.DB(), &hades.PreloadParams{
			Record: profile,
			Fields: []hades.PreloadField{
				hades.PreloadField{Name: "OwnedKeys", OrderBy: `"created_at" DESC`},
				hades.PreloadField{Name: "OwnedKeys.Game"},
			},
		})
		if err != nil {
			return errors.Wrap(err, 0)
		}

		keys := profile.OwnedKeys

		yn := &butlerd.FetchProfileOwnedKeysYieldNotification{
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

	err := sendDBKeys()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	ownedRes, err := client.ListMyOwnedKeys()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	profile.OwnedKeys = ownedRes.OwnedKeys
	err = c.Save(rc.DB(), &hades.SaveParams{
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

	res := &butlerd.FetchProfileOwnedKeysResult{}
	return res, nil
}
