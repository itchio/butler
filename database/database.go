package database

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd/horror"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/database/models/migrations"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

// Prepare synchronizes schemas, runs migrations etc.
func Prepare(consumer *state.Consumer, conn *sqlite.Conn, justCreated bool) (retErr error) {
	defer horror.RecoverInto(&retErr)

	err := models.HadesContext().AutoMigrate(conn)
	if err != nil {
		return errors.WithMessage(err, "performing automatic DB migration")
	}

	if justCreated {
		models.SetSchemaVersion(conn, migrations.LatestSchemaVersion())
	} else {
		err := migrations.Do(consumer, conn)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
