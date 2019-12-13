package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/lazyfetch"
	"github.com/itchio/butler/endpoints/fetch/pager"
	"github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"xorm.io/builder"
)

func LazyFetchProfileOwnedKeys(rc *butlerd.RequestContext, params lazyfetch.ProfiledLazyFetchParams, res lazyfetch.LazyFetchResponse) {
	consumer := rc.Consumer
	profile, client := rc.ProfileClient(params.GetProfileID())

	ft := models.FetchTargetForProfileOwnedKeys(params.GetProfileID())
	lazyfetch.Do(rc, ft, params, res, func(targets lazyfetch.Targets) {
		profile.OwnedKeys = nil
		for page := int64(1); ; page++ {
			consumer.Infof("Fetching page %d", page)

			ownedRes, err := client.ListProfileOwnedKeys(rc.Ctx, itchio.ListProfileOwnedKeysParams{
				Page: page,
			})
			models.Must(err)
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
			targets.Add(models.FetchTargetForGame(dk.Game.ID))
		}

		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustSave(conn, profile,
				hades.OmitRoot(),
				hades.AssocReplace("OwnedKeys",
					hades.Assoc("Game"),
				),
			)
		})
	})
}

func FetchProfileOwnedKeys(rc *butlerd.RequestContext, params butlerd.FetchProfileOwnedKeysParams) (*butlerd.FetchProfileOwnedKeysResult, error) {
	res := &butlerd.FetchProfileOwnedKeysResult{}

	LazyFetchProfileOwnedKeys(rc, params, res)

	rc.WithConn(func(conn *sqlite.Conn) {
		var cond builder.Cond = builder.Eq{"download_keys.owner_id": params.GetProfileID()}
		joinGames := false
		search := hades.Search{}

		switch params.SortBy {
		case "acquiredAt", "":
			search = search.OrderBy("download_keys.created_at " + pager.Ordering("DESC", params.Reverse))
		case "title":
			search = search.OrderBy("lower(games.title) " + pager.Ordering("ASC", params.Reverse))
			joinGames = true
		}

		if params.Filters.Installed {
			cond = builder.And(cond, builder.Expr("exists (select 1 from caves where caves.game_id = download_keys.game_id)"))
		}

		if params.Filters.Classification != "" {
			cond = builder.And(cond, builder.Eq{"games.classification": params.Filters.Classification})
			joinGames = true
		}

		if params.Search != "" {
			cond = builder.And(cond, builder.Like{"games.title", params.Search})
			joinGames = true
		}

		if joinGames {
			search = search.InnerJoin("games", "games.id = download_keys.game_id")
		}

		var items []*itchio.DownloadKey
		pg := pager.New(params)
		res.NextCursor = pg.Fetch(conn, &items, cond, search)
		models.MustPreload(conn, items, hades.Assoc("Game"))
		res.Items = items
	})
	return res, nil
}
