package models

import (
	"time"
)

const defaultTTL = 2 * time.Minute
const longTTL = 10 * time.Minute
const collectionGamesTTL = 7 * 24 * time.Hour

func FetchTargetForGame(gameID int64) FetchTarget {
	return FetchTarget{
		ID:   gameID,
		Type: "game",
		TTL:  defaultTTL,
	}
}

func FetchTargetForUpload(uploadID int64) FetchTarget {
	return FetchTarget{
		ID:   uploadID,
		Type: "upload",
		TTL:  defaultTTL,
	}
}

func FetchTargetForGameUploads(gameID int64) FetchTarget {
	return FetchTarget{
		ID:   gameID,
		Type: "game_uploads",
		TTL:  defaultTTL,
	}
}

func FetchTargetForUser(userID int64) FetchTarget {
	return FetchTarget{
		ID:   userID,
		Type: "user",
		TTL:  longTTL,
	}
}

func FetchTargetForProfileCollections(profileID int64) FetchTarget {
	return FetchTarget{
		ID:   profileID,
		Type: "profile_collections",
		TTL:  defaultTTL,
	}
}

func FetchTargetForProfileGames(profileID int64) FetchTarget {
	return FetchTarget{
		ID:   profileID,
		Type: "profile_games",
		TTL:  defaultTTL,
	}
}

func FetchTargetForProfileOwnedKeys(profileID int64) FetchTarget {
	return FetchTarget{
		ID:   profileID,
		Type: "profile_owned_keys",
		TTL:  defaultTTL,
	}
}

func FetchTargetForCollection(collectionID int64) FetchTarget {
	return FetchTarget{
		ID:   collectionID,
		Type: "collection",
		TTL:  defaultTTL,
	}
}

func FetchTargetForCollectionGames(collectionID int64) FetchTarget {
	return FetchTarget{
		ID:   collectionID,
		Type: "collection_games",
		TTL:  collectionGamesTTL,
	}
}

func FetchTargetForProfileOwnedBundles(profileID int64) FetchTarget {
	return FetchTarget{
		ID:   profileID,
		Type: "profile_owned_bundles",
		TTL:  defaultTTL,
	}
}

func FetchTargetForBundleGames(bundleID int64) FetchTarget {
	return FetchTarget{
		ID:   bundleID,
		Type: "bundle_games",
		TTL:  longTTL,
	}
}

func FetchTargetForProfileBundleOwnerships(profileID int64) FetchTarget {
	return FetchTarget{
		ID:   profileID,
		Type: "profile_bundle_ownerships",
		TTL:  longTTL,
	}
}
