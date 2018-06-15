package fetch

import (
	"strconv"
	"time"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func FetchProfileCollections(rc *butlerd.RequestContext, params *butlerd.FetchProfileCollectionsParams) (*butlerd.FetchProfileCollectionsResult, error) {
	consumer := rc.Consumer
	profile, client := rc.ProfileClient(params.ProfileID)

	limit := params.Limit
	if limit == 0 {
		limit = 5
	}
	consumer.Infof("Using limit of %d", limit)
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
	} else if rc.WithConnBool(ft.IsStale) {
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
			for _, ft := range fts {
				// TODO: avoid n+1
				ft.MarkFresh(conn)
			}
		})
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		var pcs []*models.ProfileCollection
		var cond builder.Cond = builder.Eq{"profile_id": profile.ID}
		var offset int64
		if params.Cursor != "" {
			if parsedOffset, err := strconv.ParseInt(params.Cursor, 10, 64); err == nil {
				offset = parsedOffset
			}
		}
		search := hades.Search().OrderBy("position ASC").Limit(limit + 1).Offset(offset)
		models.MustSelect(conn, &pcs, cond, search)
		models.MustPreload(conn, pcs, hades.Assoc("Collection"))

		for i, pc := range pcs {
			// last collection
			if i == len(pcs)-1 {
				// and we fetched more than was asked..
				if int64(len(pcs)) > limit {
					// then we have a next page
					res.NextCursor = strconv.FormatInt(offset+limit, 10)
				}
			} else {
				res.Items = append(res.Items, pc.Collection)
			}
		}
	})
	return res, nil
}
