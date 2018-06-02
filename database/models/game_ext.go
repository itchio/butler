package models

import (
	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	itchio "github.com/itchio/go-itchio"
)

// Game is defined in `go-itchio`, but helper functions are here

func GameByID(conn *sqlite.Conn, id int64) *itchio.Game {
	var g itchio.Game
	if MustSelectOne(conn, &g, builder.Eq{"id": id}) {
		return &g
	}
	return nil
}
