package models

import (
	itchio "github.com/itchio/go-itchio"
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
