package models

import itchio "github.com/itchio/go-itchio"

// Join table for Profile <has many> Games
type ProfileGame struct {
	GameID int64        `json:"gameId" gorm:"primary_key"`
	Game   *itchio.Game `json:"game,omitempty"`

	// ID of the profile this game is associated with - they're
	// not necessarily the original owner, they just have admin
	// access to it.
	ProfileID int64    `json:"profileId" gorm:"primary_key"`
	Profile   *Profile `json:"profile,omitempty"`

	// ID of the itch.io user this game actually belongs to
	UserID int64        `json:"userId"`
	User   *itchio.User `json:"user,omitempty"`

	Position int64 `json:"position"`

	// Stats

	ViewsCount     int64 `json:"viewsCount"`
	DownloadsCount int64 `json:"downloadsCount"`
	PurchasesCount int64 `json:"purchasesCount"`

	Published bool `json:"published"`
}
