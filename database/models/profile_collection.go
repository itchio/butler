package models

import itchio "github.com/itchio/go-itchio"

type ProfileCollection struct {
	CollectionID int64 `hades:"primary_key"`
	Collection   *itchio.Collection

	ProfileID int64 `hades:"primary_key"`
	Profile   *Profile

	Position int64
}
