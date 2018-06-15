package fetch

import (
	"strconv"
	"time"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func FetchProfileOwnedKeys(rc *butlerd.RequestContext, params *butlerd.FetchProfileOwnedKeysParams) (*butlerd.FetchProfileOwnedKeysResult, error) {
	consumer := rc.Consumer
	profile, client := rc.ProfileClient(params.ProfileID)

	limit := params.Limit
	if limit == 0 {
		limit = 5
	}
	consumer.Infof("Using limit of %d", limit)
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
	} else if rc.WithConnBool(ft.IsStale) {
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
			for _, ft := range fts {
				// TODO: avoid n+1
				ft.MarkFresh(conn)
			}
		})
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		var oks []*itchio.DownloadKey
		var cond builder.Cond = builder.Eq{"owner_id": profile.ID}
		var offset int64
		if params.Cursor != "" {
			if parsedOffset, err := strconv.ParseInt(params.Cursor, 10, 64); err == nil {
				offset = parsedOffset
			}
		}
		search := hades.Search().OrderBy("created_at DESC").Limit(limit + 1).Offset(offset)
		models.MustSelect(conn, &oks, cond, search)
		models.MustPreload(conn, oks, hades.Assoc("Game"))

		for i, ok := range oks {
			if i == len(oks)-1 && int64(len(oks)) > limit {
				res.NextCursor = strconv.FormatInt(offset+limit, 10)
			} else {
				res.Items = append(res.Items, ok)
			}
		}
	})
	return res, nil
}
