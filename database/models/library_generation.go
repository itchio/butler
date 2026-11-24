package models

import (
	"sync"
	"sync/atomic"

	itchio "github.com/itchio/go-itchio"
)

var libraryGeneration atomic.Uint64

// LibraryGeneration changes whenever a write may have changed which games,
// bundles and collections are in a profile's library, or their titles.
// Anything derived from the library can keep the value it was built at and
// rebuild when it differs. Read it before reading the tables, so a change
// in between is not missed.
//
// hades reports every write it makes to these tables. Raw SQL against them
// would have to call LibraryChanged itself, and so would a write inside a
// transaction, after committing: the report from hades comes before the
// commit, when a reader on another connection still sees the old rows.
func LibraryGeneration() uint64 {
	return libraryGeneration.Load()
}

func LibraryChanged() {
	libraryGeneration.Add(1)
}

// The tables behind GameInProfileLibraryCond, and behind the bundles and
// collections a profile has.
var libraryTables = sync.OnceValue(func() map[string]struct{} {
	tables := make(map[string]struct{})
	for _, model := range []any{
		&itchio.Game{},
		&itchio.DownloadKey{},
		&itchio.Bundle{},
		&itchio.BundleKey{},
		&itchio.BundleGame{},
		&itchio.Collection{},
		&itchio.CollectionGame{},
		&ProfileCollection{},
		&ProfileGame{},
		&Cave{},
	} {
		tables[hadesContext.TableName(model)] = struct{}{}
	}
	return tables
})

func isLibraryWrite(tables []string) bool {
	for _, table := range tables {
		if _, ok := libraryTables()[table]; ok {
			return true
		}
	}
	return false
}
