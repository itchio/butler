package models

import (
	"time"
)

const defaultTTL = 10 * time.Minute
const longTTL = 1 * time.Hour

func FetchTargetForGame(gameID int64) FetchTarget {
	return FetchTarget{
		ID:   gameID,
		Type: "game",
		TTL:  defaultTTL,
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
		TTL:  longTTL,
	}
}
