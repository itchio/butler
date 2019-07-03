package models

import (
	"crawshaw.io/sqlite"
	"xorm.io/builder"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
)

func DownloadKeysByGameID(conn *sqlite.Conn, gameID int64) []*itchio.DownloadKey {
	var dks []*itchio.DownloadKey
	MustSelect(conn, &dks, builder.Eq{"game_id": gameID}, hades.Search{})
	return dks
}
