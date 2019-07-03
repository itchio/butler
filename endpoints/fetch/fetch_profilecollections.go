package fetch

import (
	"crawshaw.io/sqlite"
	"xorm.io/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/lazyfetch"
	"github.com/itchio/butler/endpoints/fetch/pager"
	"github.com/itchio/hades"
)

func FetchProfileCollections(rc *butlerd.RequestContext, params butlerd.FetchProfileCollectionsParams) (*butlerd.FetchProfileCollectionsResult, error) {
	profile, client := rc.ProfileClient(params.ProfileID)

	ft := models.FetchTargetForProfileCollections(profile.ID)
	res := &butlerd.FetchProfileCollectionsResult{}

	lazyfetch.Do(rc, ft, params, res, func(targets lazyfetch.Targets) {
		collRes, err := client.ListProfileCollections(rc.Ctx)
		models.Must(err)

		profile.ProfileCollections = nil
		for i, c := range collRes.Collections {
			targets.Add(models.FetchTargetForCollection(c.ID))

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
		})
	})

	rc.WithConn(func(conn *sqlite.Conn) {
		var cond builder.Cond = builder.Eq{"profile_id": profile.ID}
		joinCollections := false
		search := hades.Search{}

		switch params.SortBy {
		case "default", "":
			search = search.OrderBy("position " + pager.Ordering("ASC", params.Reverse))
		case "updatedAt":
			search = search.OrderBy("collections.updated_at " + pager.Ordering("DESC", params.Reverse))
			joinCollections = true
		case "title":
			search = search.OrderBy("collections.title " + pager.Ordering("ASC", params.Reverse))
			joinCollections = true
		}

		if params.Search != "" {
			cond = builder.And(cond, builder.Like{"collections.title", params.Search})
			joinCollections = true
		}

		if joinCollections {
			search = search.Join("collections", "collections.id = profile_collections.collection_id")
		}

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
