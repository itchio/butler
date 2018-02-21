package models

import "time"

type CollectionGame struct {
	CollectionID int64 `json:"collectionId"`
	GameID       int64 `json:"gameId"`
	Position     int64 `json:"position"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Blurb  string `json:"blurb"`
	UserID int64  `json:"userId"`
}
