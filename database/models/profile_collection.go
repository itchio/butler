package models

import itchio "github.com/itchio/go-itchio"

type ProfileCollection struct {
	CollectionID int64 `gorm:"primary_key"`
	Collection   *itchio.Collection

	ProfileID int64 `gorm:"primary_key"`
	Profile   *Profile

	Position int64
}
