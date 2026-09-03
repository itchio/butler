package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/lazyfetch"
	"github.com/itchio/butler/endpoints/fetch/pager"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"xorm.io/builder"
)

func collectionChanged(previous, current *itchio.Collection) bool {
	if previous == nil {
		return true
	}
	if previous.GamesCount != current.GamesCount {
		return true
	}
	if previous.UpdatedAt == nil || current.UpdatedAt == nil {
		// changed if exactly one side is nil
		return previous.UpdatedAt != current.UpdatedAt
	}
	return !previous.UpdatedAt.Equal(*current.UpdatedAt)
}

func expireChangedCollectionGames(conn *sqlite.Conn, previous map[int64]*itchio.Collection, current []*itchio.Collection) {
	for _, collection := range current {
		if collectionChanged(previous[collection.ID], collection) {
			models.FetchTargetForCollectionGames(collection.ID).MustExpire(conn)
		}
	}
}

// fetchTargetList collects fetch targets outside of lazyfetch.Do
type fetchTargetList struct {
	items []models.FetchTarget
}

func (l *fetchTargetList) Add(ft models.FetchTarget) {
	l.items = append(l.items, ft)
}

func FetchProfileCollections(rc *butlerd.RequestContext, params butlerd.FetchProfileCollectionsParams) (*butlerd.FetchProfileCollectionsResult, error) {
	profile, client := rc.ProfileClient(params.ProfileID)
	ft := models.FetchTargetForProfileCollections(profile.ID)
	res := &butlerd.FetchProfileCollectionsResult{}

	// Membership of a specific game is never stored locally, only
	// carried from the API response to this reply.
	hasGame := make(map[int64]*bool)

	refresh := func(targets lazyfetch.Targets) {
		previousCollections := make(map[int64]*itchio.Collection)
		rc.WithConn(func(conn *sqlite.Conn) {
			var items []*models.ProfileCollection
			models.MustSelect(conn, &items, builder.Eq{"profile_id": profile.ID}, hades.Search{})
			models.MustPreload(conn, items, hades.Assoc("Collection"))
			for _, item := range items {
				previousCollections[item.CollectionID] = item.Collection
			}
		})

		collRes, err := client.ListProfileCollections(rc.Ctx, itchio.ListProfileCollectionsParams{
			GameID: params.GameID,
		})
		models.Must(err)

		profile.ProfileCollections = nil
		for i, c := range collRes.Collections {
			targets.Add(models.FetchTargetForCollection(c.ID))
			hasGame[c.ID] = c.HasGame

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
			expireChangedCollectionGames(conn, previousCollections, collRes.Collections)
		})
	}

	if params.GameID != 0 {
		// Always ask the API, and don't share the request with other
		// callers through lazyfetch: they wouldn't be asking about the
		// same game.
		rc.Consumer.Infof("Fetching fresh data (with membership for game %d)...", params.GameID)
		targets := &fetchTargetList{items: []models.FetchTarget{ft}}
		refresh(targets)
		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustMarkAllFresh(conn, targets.items)
		})
	} else {
		lazyfetch.Do(rc, ft, params, res, refresh)
	}

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
			search = search.OrderBy("lower(collections.title) " + pager.Ordering("ASC", params.Reverse))
			joinCollections = true
		}

		if params.Search != "" {
			cond = builder.And(cond, builder.Like{"collections.title", params.Search})
			joinCollections = true
		}

		if joinCollections {
			search = search.InnerJoin("collections", "collections.id = profile_collections.collection_id")
		}

		var items []*models.ProfileCollection
		pg := pager.New(params)
		res.NextCursor = pg.Fetch(conn, &items, cond, search)
		models.MustPreload(conn, items, hades.Assoc("Collection"))
		for _, item := range items {
			if params.GameID != 0 {
				item.Collection.HasGame = hasGame[item.CollectionID]
				if item.Collection.HasGame == nil {
					// the contract is that every item answers when a game was asked about
					notIn := false
					item.Collection.HasGame = &notIn
				}
			}
			res.Items = append(res.Items, item.Collection)
		}
	})
	return res, nil
}
