package models

import (
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
)

// AllModels contains all the tables contained in butler's database
var AllModels = []interface{}{
	&SchemaVersion{},
	&Profile{},
	&ProfileCollection{},
	&itchio.DownloadKey{},
	&itchio.Bundle{},
	&itchio.BundleGame{},
	&itchio.BundleKey{},
	&itchio.Collection{},
	&itchio.CollectionGame{},
	&ProfileGame{},
	&itchio.Game{},
	&itchio.User{},
	&Download{},
	&Cave{},
	&itchio.GameEmbedData{},
	&itchio.Sale{},
	&InstallLocation{},
	&itchio.Upload{},
	&itchio.Build{},
	&ProfileData{},
	&FetchInfo{},
	&GameUpload{},
	&CaveHistoricalPlayTime{},
	&UserGameInteraction{},
}

// declareIndexes registers secondary indexes for the game-to-profile
// lookups: bundle ownership (Fetch.GameOwnership, AccessForGameID,
// owned-bundles listing) and the library scoping in
// GameInProfileLibraryCond. bundle_games' and collection_games' composite
// primary keys already cover the (container, game) direction, and
// profile_games' primary key starts with game_id so the game-first lookup
// needs no extra index (profile-first listings like Fetch.ProfileGames
// still scan).
func declareIndexes(c *hades.Context) error {
	if err := c.DeclareIndex(&itchio.BundleKey{}, "owner_id", "bundle_id"); err != nil {
		return err
	}
	if err := c.DeclareIndex(&itchio.BundleGame{}, "game_id", "bundle_id"); err != nil {
		return err
	}
	if err := c.DeclareIndex(&itchio.CollectionGame{}, "game_id", "collection_id"); err != nil {
		return err
	}
	if err := c.DeclareIndex(&itchio.DownloadKey{}, "game_id", "owner_id"); err != nil {
		return err
	}
	return c.DeclareIndex(&Cave{}, "game_id")
}
