package fetch

import (
	"time"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/pager"
	"github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func FetchProfileOwnedKeys(rc *butlerd.RequestContext, params butlerd.FetchProfileOwnedKeysParams) (*butlerd.FetchProfileOwnedKeysResult, error) {
	consumer := rc.Consumer
	profile, client := rc.ProfileClient(params.ProfileID)

	ft := models.FetchTarget{
		Type: "profile_owned_keys",
		ID:   profile.ID,
		TTL:  10 * time.Minute,
	}

	fresh := false
	res := &butlerd.FetchProfileOwnedKeysResult{}

	if params.Fresh {
		consumer.Infof("Doing remote fetch (Fresh specified)")
		fresh = true
	} else if rc.WithConnBool(ft.MustIsStale) {
		consumer.Infof("Returning stale results")
		res.Stale = true
	}

	if fresh {
		fts := []models.FetchTarget{ft}

		profile.OwnedKeys = nil
		for page := int64(1); ; page++ {
			consumer.Infof("Fetching page %d", page)

			ownedRes, err := client.ListProfileOwnedKeys(itchio.ListProfileOwnedKeysParams{
				Page: page,
			})
			if err != nil {
				return nil, errors.WithStack(err)
			}
			numPageItems := int64(len(ownedRes.OwnedKeys))

			if numPageItems == 0 {
				break
			}

			profile.OwnedKeys = append(profile.OwnedKeys, ownedRes.OwnedKeys...)
			rc.WithConn(func(conn *sqlite.Conn) {
				models.MustSave(conn, profile,
					hades.OmitRoot(),
					hades.Assoc("OwnedKeys",
						hades.Assoc("Game"),
					),
				)
			})
		}

		for _, dk := range profile.OwnedKeys {
			fts = append(fts, models.FetchTarget{
				Type: "game",
				ID:   dk.Game.ID,
				TTL:  10 * time.Minute,
			})
		}

		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustSave(conn, profile,
				hades.OmitRoot(),
				hades.AssocReplace("OwnedKeys",
					hades.Assoc("Game"),
				),
			)
			models.MustMarkAllFresh(conn, fts)
		})
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		var cond builder.Cond = builder.Eq{"owner_id": profile.ID}
		search := hades.Search{}.OrderBy("created_at DESC")

		var items []*itchio.DownloadKey
		pg := pager.New(params)
		res.NextCursor = pg.Fetch(conn, &items, cond, search)
		models.MustPreload(conn, items, hades.Assoc("Game"))
		res.Items = items
	})
	return res, nil
}
