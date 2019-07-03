package models

import (
	"crawshaw.io/sqlite"
	"xorm.io/builder"
	itchio "github.com/itchio/go-itchio"
)

// User is defined in `go-itchio`, but helper functions are here

func UserByID(conn *sqlite.Conn, id int64) *itchio.User {
	var u itchio.User
	if MustSelectOne(conn, &u, builder.Eq{"id": id}) {
		return &u
	}
	return nil
}
