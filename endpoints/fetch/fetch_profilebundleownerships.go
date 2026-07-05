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
			}
		})

		rc.WithConn(func(conn *sqlite.Conn) {
			// Counts and staleness come from the per-bundle targets rather
			// than the loop above: a concurrent caller shares the
			// singleflight without running the task, and a truncated walk
			// leaves its bundle target stale even though the loop completed.
			summarizeBundleOwnerships(conn, profile.ID, ft, res)
			if res.Stale {
				// Fetch.GameOwnership trusts the profile-level target as
				// "membership index is complete"; don't leave it fresh over
				// an incomplete sync.
				ft.MustExpire(conn)
			}
		})

		return res, nil
	}

	// Fresh:false - report cached status only.
	rc.WithConn(func(conn *sqlite.Conn) {
		summarizeBundleOwnerships(conn, profile.ID, ft, res)
	})

	return res, nil
}

// summarizeBundleOwnerships fills res with the profile's bundle sync status
// as recorded by fetch targets: total distinct owned bundles, how many have
// a fresh bundle_games target, and whether anything involved is stale.
func summarizeBundleOwnerships(conn *sqlite.Conn, profileID int64, ft models.FetchTarget, res *butlerd.FetchProfileBundleOwnershipsResult) {
	bundleIDs := models.DistinctOwnedBundleIDs(conn, profileID)
	res.TotalBundles = int64(len(bundleIDs))

	stale := models.FetchTargetForProfileOwnedBundles(profileID).MustIsStale(conn) ||
		ft.MustIsStale(conn)
	var synced int64
	for _, bundleID := range bundleIDs {
		if models.FetchTargetForBundleGames(bundleID).MustIsStale(conn) {
			stale = true
		} else {
			synced++
		}
	}
	res.SyncedBundles = synced
	res.Stale = stale
}
