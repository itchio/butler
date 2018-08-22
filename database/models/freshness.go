package models

import (
	"time"
)

const defaultTTL = 30 * time.Minute
const longTTL = 2 * time.Hour

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
		TTL:  longTTL,
	}
}
