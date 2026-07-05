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
}

// declareIndexes registers secondary indexes for the bundle ownership
// lookups (Fetch.GameOwnership, AccessForGameID, owned-bundles listing).
// bundle_games' composite primary key already covers the (bundle_id,
// game_id) direction.
func declareIndexes(c *hades.Context) error {
	if err := c.DeclareIndex(&itchio.BundleKey{}, "owner_id", "bundle_id"); err != nil {
		return err
	}
	return c.DeclareIndex(&itchio.BundleGame{}, "game_id", "bundle_id")
}
