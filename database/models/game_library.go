package models

import (
	"xorm.io/builder"
)

// GameInProfileLibraryCond restricts a games query to the profile's library.
// The games table is a cache shared by all profiles and filled by any fetch
// (including browsing a game page), so membership comes from the relation
// tables instead: owned via download key, part of an owned bundle, in one of
// the profile's collections, or on the profile's dashboard. Installs (caves)
// have no profile, so installed games match for every profile.
//
// Every branch is correlated on games.id and backed by a game-first index
// from declareIndexes. The bundle branch uses cross join to pin bundle_games
// as the driving table: butler never runs ANALYZE, and without stats SQLite
// drives from bundle_keys and probes every owned bundle per candidate row.
func GameInProfileLibraryCond(profileID int64) builder.Cond {
	return builder.Or(
		builder.Expr(
			"exists (select 1 from download_keys where download_keys.game_id = games.id and download_keys.owner_id = ?)",
			profileID,
		),
		builder.Expr(
			"exists (select 1 from bundle_games cross join bundle_keys on bundle_keys.bundle_id = bundle_games.bundle_id where bundle_games.game_id = games.id and bundle_keys.owner_id = ?)",
			profileID,
		),
		builder.Expr(
			"exists (select 1 from collection_games join profile_collections on profile_collections.collection_id = collection_games.collection_id where collection_games.game_id = games.id and profile_collections.profile_id = ?)",
			profileID,
		),
		builder.Expr(
			"exists (select 1 from profile_games where profile_games.game_id = games.id and profile_games.profile_id = ?)",
			profileID,
		),
		builder.Expr("exists (select 1 from caves where caves.game_id = games.id)"),
	)
}
