package models

import (
	itchio "github.com/itchio/go-itchio"
	"github.com/jinzhu/gorm"
)

type DownloadKey struct {
	ID int64 `json:"id"`

	GameID int64        `json:"gameId"`
	Game   *itchio.Game `json:"game,omitempty"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`

	OwnerID int64        `json:"ownerId"`
	Owner   *itchio.User `json:"owner,omitempty"`
}

func DownloadKeysByGameID(db *gorm.DB, gameID int64) []*DownloadKey {
	var keys []*DownloadKey
	err := db.Where("game_id = ?", gameID).Find(&keys).Error
	if err != nil {
		panic(err)
	}
	return keys
}
