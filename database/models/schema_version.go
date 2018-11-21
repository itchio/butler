package models

import (
	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
)

type SchemaVersion struct {
	ID      string
	Version int64
}

const SchemaVersionID = "database"

func GetSchemaVersion(conn *sqlite.Conn) int64 {
	var sv SchemaVersion
	MustSelectOne(conn, &sv, builder.Eq{"id": SchemaVersionID})
	return sv.Version
}

func SetSchemaVersion(conn *sqlite.Conn, version int64) {
	sv := &SchemaVersion{
		ID:      SchemaVersionID,
		Version: version,
	}
	MustSave(conn, sv)
}
