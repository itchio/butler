package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
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
		collRes, err := client.ListProfileCollections()
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
