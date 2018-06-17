package fetch

import (
	"time"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/pager"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func FetchProfileCollections(rc *butlerd.RequestContext, params *butlerd.FetchProfileCollectionsParams) (*butlerd.FetchProfileCollectionsResult, error) {
	consumer := rc.Consumer
	profile, client := rc.ProfileClient(params.ProfileID)

	ft := models.FetchTarget{
		Type: "profile_collections",
		ID:   profile.ID,
		TTL:  10 * time.Minute,
	}

	fresh := false
	res := &butlerd.FetchProfileCollectionsResult{}

	if params.Fresh {
		consumer.Infof("Doing remote fetch (Fresh specified)")
		fresh = true
	} else if rc.WithConnBool(ft.MustIsStale) {
		consumer.Infof("Returning stale results")
		res.Stale = true
	}

	if fresh {
		fts := []models.FetchTarget{ft}

		collRes, err := client.ListProfileCollections()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		consumer.Statf("Retrieved %d collections", len(collRes.Collections))

		profile.ProfileCollections = nil
		for i, c := range collRes.Collections {
			fts = append(fts, models.FetchTarget{
				Type: "collection",
				ID:   c.ID,
				TTL:  10 * time.Minute,
			})

			profile.ProfileCollections = append(profile.ProfileCollections, &models.ProfileCollection{
				// Other fields are set when saving the association
				Collection: c,
				Position:   int64(i),
			})
		}

		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustSave(conn, profile,
				hades.AssocReplace("ProfileCollections",
					hades.Assoc("Collection"),
				),
			)
			models.MustMarkAllFresh(conn, fts)
		})
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		var cond builder.Cond = builder.Eq{"profile_id": profile.ID}
		search := hades.Search{}.OrderBy("position ASC")

		var items []*models.ProfileCollection
		pg := pager.New(params)
		res.NextCursor = pg.Fetch(conn, &items, cond, search)
		models.MustPreload(conn, items, hades.Assoc("Collection"))
		for _, item := range items {
			res.Items = append(res.Items, item.Collection)
		}
	})
	return res, nil
}
