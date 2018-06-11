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

	doRemoteFetch := false

	if params.IgnoreCache {
		consumer.Infof("Doing remote fetch (IgnoreCache specified)")
		doRemoteFetch = true
	} else if rc.WithConnBool(ft.IsStale) {
		consumer.Infof("Doing remote fetch (Is stale)")
		doRemoteFetch = true
	}

	if doRemoteFetch {
		collRes, err := client.ListProfileCollections()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		consumer.Statf("Retrieved %d collections", len(collRes.Collections))

		profile.ProfileCollections = nil
		for i, c := range collRes.Collections {
			profile.ProfileCollections = append(profile.ProfileCollections, &models.ProfileCollection{
				// Other fields are set when saving the association
				Collection: c,
				Position:   int64(i),
			})
		}

		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustSave(conn, profile,
				hades.Assoc("ProfileCollections",
					hades.Assoc("Collection"),
				),
			)
			ft.MarkFresh(conn)
		})
	}

	res := &butlerd.FetchProfileCollectionsResult{}
	var pcs []*models.ProfileCollection
	rc.WithConn(func(conn *sqlite.Conn) {
		var cond builder.Cond = builder.Eq{"profile_id": profile.ID}
		if params.Cursor != "" {
			cond = builder.And(cond, builder.Gte{"position": params.Cursor})
		}
		models.MustSelect(conn, &pcs, cond, hades.Search().OrderBy("position ASC").Limit(limit+1))
		models.MustPreload(conn, pcs, hades.Assoc("Collection"))

		for i, pc := range pcs {
			if i == len(pcs)-1 {
				res.NextCursor = strconv.FormatInt(pc.Position, 10)
			} else {
				res.Items = append(res.Items, pc.Collection)
			}
		}
	})
	return res, nil
}
