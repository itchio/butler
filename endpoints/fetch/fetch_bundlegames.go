package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/lazyfetch"
	"github.com/itchio/butler/endpoints/fetch/pager"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
	"xorm.io/builder"
)

func LazyFetchBundleGames(rc *butlerd.RequestContext, params lazyfetch.ProfiledLazyFetchParams, res lazyfetch.LazyFetchResponse, bundleID int64) {
	ft := models.FetchTargetForBundleGames(bundleID)
	truncated := false
	lazyfetch.Do(rc, ft, params, res, func(targets lazyfetch.Targets) {
		_, client := rc.ProfileClient(params.GetProfileID())

		var fakeBundle = &itchio.Bundle{
			ID: bundleID,
		}
		// The full membership is accumulated in memory for the final
		// AssocReplace; this is bounded by bundle size (a few thousand
		// rows for the largest bundles).
		var bundleGames []*itchio.BundleGame

		for page := int64(1); ; page++ {
			rc.Consumer.Infof("Fetching bundle %d page %d", bundleID, page)

			gamesRes, err := client.GetBundleGames(rc.Ctx, itchio.GetBundleGamesParams{
				BundleID: bundleID,
				Page:     page,
			})
			models.Must(err)
			numPageGames := int64(len(gamesRes.BundleGames))

			if numPageGames == 0 {
				break
			}

			bundleGames = append(bundleGames, gamesRes.BundleGames...)

			rc.WithConn(func(conn *sqlite.Conn) {
				// Save only this page incrementally to avoid reprocessing all previous pages.
				fakeBundle.BundleGames = gamesRes.BundleGames
				models.MustSave(conn, fakeBundle,
					hades.OmitRoot(),
					hades.Assoc("BundleGames",
						hades.Assoc("Game",
							hades.Assoc("Sale"),
						),
					),
				)
			})

			for _, bg := range gamesRes.BundleGames {
				if bg.Game != nil {
					targets.Add(models.FetchTargetForGame(bg.Game.ID))
				}
			}
		}

		rc.WithConn(func(conn *sqlite.Conn) {
			fakeBundle.BundleGames = bundleGames
			models.MustSave(conn, fakeBundle,
				hades.OmitRoot(),
				hades.AssocReplace("BundleGames"),
			)

			// Ownership inference depends on complete membership: if the
			// walk came up far short of the bundle's advertised games_count,
			// treat the sync as truncated rather than trusting it for a full
			// TTL. Small drift is normal (games_count can include delisted
			// games), so only large gaps count.
			var bundle itchio.Bundle
			if models.MustSelectOne(conn, &bundle, builder.Eq{"id": bundleID}) && bundle.GamesCount > 0 {
				fetched := int64(len(bundleGames))
				missing := bundle.GamesCount - fetched
				if missing > 0 {
					tolerance := bundle.GamesCount / 20
					if tolerance < 5 {
						tolerance = 5
					}
					if missing > tolerance {
						truncated = true
					}
					rc.Consumer.Warnf(
						"Bundle %d: fetched %d games, expected %d (missing %d, tolerance %d)",
						bundleID, fetched, bundle.GamesCount, missing, tolerance,
					)
				}
			}
		})
	})

	if truncated {
		// leave the target stale so the next sync retries, instead of
		// serving a truncated membership set as fresh for a whole TTL
		rc.WithConn(func(conn *sqlite.Conn) {
			ft.MustExpire(conn)
		})
	}
}

func FetchBundleGames(rc *butlerd.RequestContext, params butlerd.FetchBundleGamesParams) (*butlerd.FetchBundleGamesResult, error) {
	if params.BundleID == 0 {
		return nil, errors.New("bundleId must be non-zero")
	}
	res := &butlerd.FetchBundleGamesResult{}
	LazyFetchBundleGames(rc, params, res, params.BundleID)

	rc.WithConn(func(conn *sqlite.Conn) {
		var cond builder.Cond = builder.Eq{"bundle_id": params.BundleID}
		joinGames := false
		search := hades.Search{}

		switch params.SortBy {
		case "default", "":
			search = search.OrderBy("position " + pager.Ordering("ASC", params.Reverse))
		case "title":
			search = search.OrderBy("lower(games.title) " + pager.Ordering("ASC", params.Reverse))
			joinGames = true
		}
		// game_id tiebreak keeps pagination stable if positions or titles repeat
		search = search.OrderBy("game_id ASC")

		if params.Filters.Installed {
			cond = builder.And(cond, builder.Expr("exists (select 1 from caves where caves.game_id = bundle_games.game_id)"))
		}

		if params.Filters.Classification != "" {
			cond = builder.And(cond, builder.Eq{"games.classification": params.Filters.Classification})
			joinGames = true
		}

		if pc := condForPlatformFilter(params.Filters.Platform); pc != nil {
			cond = builder.And(cond, pc)
			joinGames = true
		}

		if params.Search != "" {
			cond = builder.And(cond, builder.Like{"games.title", params.Search})
			joinGames = true
		}

		if joinGames {
			search = search.InnerJoin("games", "games.id = bundle_games.game_id")
		}

		var items []*itchio.BundleGame
		pg := pager.New(params)
		res.NextCursor = pg.Fetch(conn, &items, cond, search)
		models.MustPreload(conn, items, hades.Assoc("Game"))
		for _, item := range items {
			if item.Game == nil {
				// BundleGame.game is a required field; skip rows
				// whose game record is missing
				continue
			}
			res.Items = append(res.Items, item)
		}
	})

	return res, nil
}
