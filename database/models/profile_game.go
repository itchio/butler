package models

import (
	itchio "github.com/itchio/go-itchio"
	"github.com/jinzhu/gorm"
)

// Join table for Profile <has many> Games
type ProfileGame struct {
	GameID int64        `json:"gameId" gorm:"primary_key;auto_increment:false"`
	Game   *itchio.Game `json:"game,omitempty"`

	// ID of the profile this game is associated with - they're
	// not necessarily the original owner, they just have admin
	// access to it.
	ProfileID int64    `json:"profileId" gorm:"primary_key;auto_increment:false"`
	Profile   *Profile `json:"profile,omitempty"`

	Position int64 `json:"position"`

	// Stats

	ViewsCount     int64 `json:"viewsCount"`
	DownloadsCount int64 `json:"downloadsCount"`
	PurchasesCount int64 `json:"purchasesCount"`

	Published bool `json:"published"`
}

func ProfileGamesByGameID(db *gorm.DB, gameID int64) []*ProfileGame {
	var pgs []*ProfileGame
	err := db.Where("game_id = ?", gameID).Find(&pgs).Error
	if err != nil {
		panic(err)
	}

	return pgs
}
