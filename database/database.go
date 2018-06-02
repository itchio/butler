package database

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/database/models"
	"github.com/pkg/errors"
)

// Prepare synchronizes schemas, runs migrations etc.
func Prepare(conn *sqlite.Conn) error {
	err := models.HadesContext().AutoMigrate(conn)
	if err != nil {
		return errors.WithMessage(err, "performing automatic DB migration")
	}

	return nil
}
