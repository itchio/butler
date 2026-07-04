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

func FetchProfileOwnedBundles(rc *butlerd.RequestContext, params butlerd.FetchProfileOwnedBundlesParams) (*butlerd.FetchProfileOwnedBundlesResult, error) {
	profile, client := rc.ProfileClient(params.ProfileID)
	ft := models.FetchTargetForProfileOwnedBundles(profile.ID)
	res := &butlerd.FetchProfileOwnedBundlesResult{}

	lazyfetch.Do(rc, ft, params, res, func(targets lazyfetch.Targets) {
		bundleKeysRes, err := client.ListProfileOwnedBundles(rc.Ctx)
		models.Must(err)

		// Null out inline BundleGames so the locally-paginated bundle_games
		// table is not clobbered by partial inline data.
		for _, bk := range bundleKeysRes.BundleKeys {
			if bk.Bundle != nil {
				bk.Bundle.BundleGames = nil
			}
		}

		fakeProfile := &models.Profile{ID: profile.ID}
		fakeProfile.BundleKeys = bundleKeysRes.BundleKeys
		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustSave(conn, fakeProfile,
				hades.OmitRoot(),
				hades.AssocReplace("BundleKeys",
					hades.Assoc("Bundle"),
				),
			)
		})
	})

	rc.WithConn(func(conn *sqlite.Conn) {
		// Dedupe by bundle_id: keep the newest bundle_key (max created_at,
		// tie-break by max id) for each distinct bundle owned by the profile.
		dedupedExpr := builder.Expr(
			"bundle_keys.id in (" +
				"select id from bundle_keys bk2 " +
				"where bk2.owner_id = bundle_keys.owner_id " +
				"and bk2.bundle_id = bundle_keys.bundle_id " +
				"order by bk2.created_at desc, bk2.id desc limit 1" +
				")",
		)
		cond := builder.And(
			builder.Eq{"bundle_keys.owner_id": params.ProfileID},
			dedupedExpr,
		)
		joinBundles := false
		search := hades.Search{}

		switch params.SortBy {
		case "acquiredAt", "":
			search = search.OrderBy("bundle_keys.created_at " + pager.Ordering("DESC", params.Reverse))
		case "title":
			search = search.OrderBy("lower(bundles.title) " + pager.Ordering("ASC", params.Reverse))
			joinBundles = true
		case "updatedAt":
			search = search.OrderBy("bundles.updated_at " + pager.Ordering("DESC", params.Reverse))
			joinBundles = true
		case "gamesCount":
			search = search.OrderBy("bundles.games_count " + pager.Ordering("DESC", params.Reverse))
			joinBundles = true
		}

		if params.Search != "" {
			cond = builder.And(cond, builder.Like{"bundles.title", params.Search})
			joinBundles = true
		}

		if joinBundles {
			search = search.InnerJoin("bundles", "bundles.id = bundle_keys.bundle_id")
		}

		var items []*itchio.BundleKey
		pg := pager.New(params)
		res.NextCursor = pg.Fetch(conn, &items, cond, search)
		models.MustPreload(conn, items, hades.Assoc("Bundle"))
		res.Items = items
	})

	return res, nil
}
