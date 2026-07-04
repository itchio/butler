package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"xorm.io/builder"
)

func FetchGameOwnership(rc *butlerd.RequestContext, params butlerd.FetchGameOwnershipParams) (*butlerd.FetchGameOwnershipResult, error) {
	res := &butlerd.FetchGameOwnershipResult{}

	rc.WithConn(func(conn *sqlite.Conn) {
		var dk itchio.DownloadKey
		if models.MustSelectOne(conn, &dk,
			builder.And(
				builder.Eq{"download_keys.owner_id": params.ProfileID},
				builder.Eq{"download_keys.game_id": params.GameID},
			),
		) {
			res.Owned = true
			res.Source = "download_key"
			res.DownloadKeyID = dk.ID
			return
		}

		bundleID := models.BundleIDOwningGameForProfile(conn, params.GameID, params.ProfileID)

		ownedBundlesStale := models.FetchTargetForProfileOwnedBundles(params.ProfileID).MustIsStale(conn)
		bundleOwnershipsStale := models.FetchTargetForProfileBundleOwnerships(params.ProfileID).MustIsStale(conn)

		if bundleID != 0 {
			res.Owned = true
			res.Source = "bundle"
			res.BundleID = bundleID
			if ownedBundlesStale || bundleOwnershipsStale {
				res.Stale = true
			}
			return
		}

		// No local ownership. If the bundle ownership index is stale, the
		// negative answer may not yet reflect newly-purchased bundles.
		// Deliberately not tied to the owned-keys target: its short TTL
		// would make almost every negative answer stale, and clients react
		// to stale by triggering the profile-wide bundle sync. Materialized
		// download keys owned by this profile were already checked above,
		// and direct-purchase ownership is the renderer's commons' job.
		if ownedBundlesStale || bundleOwnershipsStale {
			res.Stale = true
		}
	})

	return res, nil
}
