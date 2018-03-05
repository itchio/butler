package models

import itchio "github.com/itchio/go-itchio"

type ProfileCollection struct {
	CollectionID int64 `gorm:"primary_key;auto_increment:false"`
	Collection   *itchio.Collection

	ProfileID int64 `gorm:"primary_key;auto_increment:false"`
	Profile   *Profile

	Position int64
}
