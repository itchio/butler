package models

import (
	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	itchio "github.com/itchio/go-itchio"
)

func DownloadKeysByGameID(conn *sqlite.Conn, gameID int64) []*itchio.DownloadKey {
	var dks []*itchio.DownloadKey
	MustSelect(conn, &dks, builder.Eq{"game_id": gameID}, nil)
	return dks
}
