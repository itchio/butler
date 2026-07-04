package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/lazyfetch"
)

func FetchProfileBundleOwnerships(rc *butlerd.RequestContext, params butlerd.FetchProfileBundleOwnershipsParams) (*butlerd.FetchProfileBundleOwnershipsResult, error) {
	profile, _ := rc.ProfileClient(params.ProfileID)
	res := &butlerd.FetchProfileBundleOwnershipsResult{}

	ft := models.FetchTargetForProfileBundleOwnerships(profile.ID)

	if params.Fresh {
		// Refresh the owned-bundles list first; this is small and tells us
		// which bundle_games targets to sync.
		_, err := FetchProfileOwnedBundles(rc, butlerd.FetchProfileOwnedBundlesParams{
			ProfileID: profile.ID,
			Fresh:     true,
		})
		if err != nil {
			return nil, err
		}

		var bundleIDs []int64
		rc.WithConn(func(conn *sqlite.Conn) {
			bundleIDs = models.DistinctOwnedBundleIDs(conn, profile.ID)
		})

		res.TotalBundles = int64(len(bundleIDs))

		lazyfetch.Do(rc, ft, params, res, func(targets lazyfetch.Targets) {
			for _, bundleID := range bundleIDs {
				// Ensure this bundle's membership is fresh, respecting the
				// per-bundle TTL: only bundles whose local game list is
				// stale get re-walked, so a profile-wide fresh sync stays
				// cheap when most bundles are already synced.
				bgParams := butlerd.FetchBundleGamesParams{
					ProfileID: profile.ID,
					BundleID:  bundleID,
				}
				bgRes := &butlerd.FetchBundleGamesResult{}
				LazyFetchBundleGames(rc, bgParams, bgRes, bundleID)
				if bgRes.Stale {
					bgParams.Fresh = true
					bgRes = &butlerd.FetchBundleGamesResult{}
					LazyFetchBundleGames(rc, bgParams, bgRes, bundleID)
				}
				res.SyncedBundles++
			}
		})

		return res, nil
	}

	// Fresh:false - report cached status only.
	rc.WithConn(func(conn *sqlite.Conn) {
		bundleIDs := models.DistinctOwnedBundleIDs(conn, profile.ID)
		res.TotalBundles = int64(len(bundleIDs))

		stale := false
		if models.FetchTargetForProfileOwnedBundles(profile.ID).MustIsStale(conn) {
			stale = true
		}
		if ft.MustIsStale(conn) {
			stale = true
		}
		var synced int64
		for _, bundleID := range bundleIDs {
			if !models.FetchTargetForBundleGames(bundleID).MustIsStale(conn) {
				synced++
			} else {
				stale = true
			}
		}
		res.SyncedBundles = synced
		res.Stale = stale
	})

	return res, nil
}
