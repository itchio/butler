package models

import (
	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	itchio "github.com/itchio/go-itchio"
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

func DownloadKeysByGameID(conn *sqlite.Conn, gameID int64) []*DownloadKey {
	var keys []*DownloadKey
	MustSelect(conn, &keys, builder.Eq{"game_id": gameID}, nil)
	return keys
}
