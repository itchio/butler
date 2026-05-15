package models

import (
	"crawshaw.io/sqlite"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"xorm.io/builder"
)

// BundleKeysByGameID returns all bundle keys held by any profile that grant
// access to the given game via bundle membership. May return multiple keys
// for the same profile/bundle if a bundle was purchased more than once.
func BundleKeysByGameID(conn *sqlite.Conn, gameID int64) []*itchio.BundleKey {
	var bks []*itchio.BundleKey
	MustSelect(conn, &bks,
		builder.Expr(
			"exists (select 1 from bundle_games where bundle_games.bundle_id = bundle_keys.bundle_id and bundle_games.game_id = ?)",
			gameID,
		),
		hades.Search{},
	)
	return bks
}

// ProfileOwnsGameViaBundle reports whether the given profile owns a bundle
// that contains the given game.
func ProfileOwnsGameViaBundle(conn *sqlite.Conn, profileID int64, gameID int64) bool {
	count := MustCount(conn, &itchio.BundleKey{},
		builder.And(
			builder.Eq{"bundle_keys.owner_id": profileID},
			builder.Expr(
				"exists (select 1 from bundle_games where bundle_games.bundle_id = bundle_keys.bundle_id and bundle_games.game_id = ?)",
				gameID,
			),
		),
	)
	return count > 0
}

// BundleIDOwningGameForProfile returns one bundle ID, owned by the given
// profile, that contains the given game. Returns 0 if none.
func BundleIDOwningGameForProfile(conn *sqlite.Conn, gameID int64, profileID int64) int64 {
	var bk itchio.BundleKey
	if MustSelectOne(conn, &bk,
		builder.And(
			builder.Eq{"bundle_keys.owner_id": profileID},
			builder.Expr(
				"exists (select 1 from bundle_games where bundle_games.bundle_id = bundle_keys.bundle_id and bundle_games.game_id = ?)",
				gameID,
			),
		),
	) {
		return bk.BundleID
	}
	return 0
}

// BundleIDOwningGameAnyProfile returns one (bundleID, profileID) pair for the
// given game from any locally-known bundle key. Used as a fallback for
// callers that do not know which profile to materialize as.
func BundleIDOwningGameAnyProfile(conn *sqlite.Conn, gameID int64) (bundleID int64, profileID int64) {
	var bk itchio.BundleKey
	if MustSelectOne(conn, &bk,
		builder.Expr(
			"exists (select 1 from bundle_games where bundle_games.bundle_id = bundle_keys.bundle_id and bundle_games.game_id = ?)",
			gameID,
		),
	) {
		return bk.BundleID, bk.OwnerID
	}
	return 0, 0
}

// DistinctOwnedBundleIDs returns the distinct bundle IDs owned by the given
// profile (deduping across multiple purchases of the same bundle).
func DistinctOwnedBundleIDs(conn *sqlite.Conn, profileID int64) []int64 {
	var ids []int64
	MustExecRaw(conn,
		"select distinct bundle_id from bundle_keys where owner_id = ?",
		func(stmt *sqlite.Stmt) error {
			ids = append(ids, stmt.ColumnInt64(0))
			return nil
		},
		profileID,
	)
	return ids
}
