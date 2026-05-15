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
	lazyfetch.Do(rc, ft, params, res, func(targets lazyfetch.Targets) {
		_, client := rc.ProfileClient(params.GetProfileID())

		var fakeBundle = &itchio.Bundle{
			ID: bundleID,
		}
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
		})
	})
}

func FetchBundleGames(rc *butlerd.RequestContext, params butlerd.FetchBundleGamesParams) (*butlerd.FetchBundleGamesResult, error) {
	if params.BundleID == 0 {
		return nil, errors.New("bundleId must be non-zero")
	}
	res := &butlerd.FetchBundleGamesResult{}
	LazyFetchBundleGames(rc, params, res, params.BundleID)

	rc.WithConn(func(conn *sqlite.Conn) {
		cond := builder.Eq{"bundle_id": params.BundleID}
		search := hades.Search{}.OrderBy("position " + pager.Ordering("ASC", false))

		var items []*itchio.BundleGame
		pg := pager.New(params)
		res.NextCursor = pg.Fetch(conn, &items, cond, search)
		models.MustPreload(conn, items, hades.Assoc("Game"))
		res.Items = items
	})

	return res, nil
}
