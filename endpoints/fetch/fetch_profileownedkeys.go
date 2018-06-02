package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func FetchProfileOwnedKeys(rc *butlerd.RequestContext, params *butlerd.FetchProfileOwnedKeysParams) (*butlerd.FetchProfileOwnedKeysResult, error) {
	profile, client := rc.ProfileClient(params.ProfileID)

	sendDBKeys := func() error {
		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustPreload(conn, &hades.PreloadParams{
				Record: profile,
				Fields: []hades.PreloadField{
					{Name: "OwnedKeys", Search: hades.Search().OrderBy("created_at DESC")},
					{Name: "OwnedKeys.Game"},
				},
			})
		})

		keys := profile.OwnedKeys

		yn := &butlerd.FetchProfileOwnedKeysYieldNotification{
			Offset: 0,
			Total:  int64(len(keys)),
			Items:  keys,
		}
		err := messages.FetchProfileOwnedKeysYield.Notify(rc, yn)
		if err != nil {
			return errors.WithStack(err)
		}

		return nil
	}

	err := sendDBKeys()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ownedRes, err := client.ListProfileOwnedKeys()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	profile.OwnedKeys = ownedRes.OwnedKeys
	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustSave(conn, &hades.SaveParams{
			Record: profile,
			Assocs: []string{"OwnedKeys"},
		})
	})

	err = sendDBKeys()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := &butlerd.FetchProfileOwnedKeysResult{}
	return res, nil
}
