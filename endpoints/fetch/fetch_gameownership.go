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
		gameOwnershipFromConn(conn, params.ProfileID, params.GameID, res)
	})
	return res, nil
}

// gameOwnershipFromConn answers ownership for one (profile, game) pair from
// local data only: materialized download keys first, then bundle membership.
func gameOwnershipFromConn(conn *sqlite.Conn, profileID int64, gameID int64, res *butlerd.FetchGameOwnershipResult) {
	var dk itchio.DownloadKey
	if models.MustSelectOne(conn, &dk,
		builder.And(
			builder.Eq{"download_keys.owner_id": profileID},
			builder.Eq{"download_keys.game_id": gameID},
		),
	) {
		res.Owned = true
		res.Source = "download_key"
		res.DownloadKeyID = dk.ID
		return
	}

	bundleID := models.BundleIDOwningGameForProfile(conn, gameID, profileID)

	ownedBundlesStale := models.FetchTargetForProfileOwnedBundles(profileID).MustIsStale(conn)
	bundleOwnershipsStale := models.FetchTargetForProfileBundleOwnerships(profileID).MustIsStale(conn)

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
}
