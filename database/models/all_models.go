package models

import (
	itchio "github.com/itchio/go-itchio"
)

// AllModels contains all the tables contained in butler's database
var AllModels = []interface{}{
	&Profile{},
	&ProfileCollection{},
	&itchio.DownloadKey{},
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
}
