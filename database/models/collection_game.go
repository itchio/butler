package models

import (
	"time"

	itchio "github.com/itchio/go-itchio"
)

type CollectionGame struct {
	CollectionID int64 `json:"collectionId"`
	Collection   *itchio.Collection

	GameID int64 `json:"gameId"`
	Game   *itchio.Game

	Position int64 `json:"position"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Blurb  string `json:"blurb"`
	UserID int64  `json:"userId"`
}
