package models

import (
	"crawshaw.io/sqlite"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"xorm.io/builder"
)

// Join table for Profile <has many> Games
type ProfileGame struct {
	GameID int64        `json:"gameId" hades:"primary_key"`
	Game   *itchio.Game `json:"game,omitempty"`

	// ID of the profile this game is associated with - they're
	// not necessarily the original owner, they just have admin
	// access to it.
	ProfileID int64    `json:"profileId" hades:"primary_key"`
	Profile   *Profile `json:"profile,omitempty"`

	Position int64 `json:"position"`

	// Stats

	ViewsCount     int64 `json:"viewsCount"`
	DownloadsCount int64 `json:"downloadsCount"`
	PurchasesCount int64 `json:"purchasesCount"`

	Published bool `json:"published"`
}

func ProfileGamesByGameID(conn *sqlite.Conn, gameID int64) []*ProfileGame {
	var pgs []*ProfileGame
	MustSelect(conn, &pgs, builder.Eq{"game_id": gameID}, hades.Search{})
	return pgs
}
