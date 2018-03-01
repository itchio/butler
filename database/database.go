package database

import (
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/jinzhu/gorm"
	// enable sqlite3 dialect for gorm
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var debugSql = os.Getenv("BUTLER_SQL") == "1"

var _db *gorm.DB

// Models contains all the tables contained in butler's database
var Models = []interface{}{
	&models.Profile{},
	&itchio.DownloadKey{},
	&itchio.Collection{},
	&models.CollectionGame{},
	&models.DashboardGame{},
	&itchio.Game{},
	&itchio.User{},
	&models.Download{},
	&models.Cave{},
	&itchio.GameEmbedData{},
	&itchio.Sale{},
}

// OpenAndPrepare returns a connection to butler's sqlite database
func OpenAndPrepare(dbPath string) (*gorm.DB, error) {
	err := os.MkdirAll(filepath.Dir(dbPath), 0755)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	db, err := gorm.Open("sqlite3", dbPath)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return Prepare(db)
}

// Prepare synchronizes schemas, runs migrations etc.
func Prepare(db *gorm.DB) (*gorm.DB, error) {
	err := db.AutoMigrate(Models...).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if debugSql {
		db.LogMode(true)
	}

	return db, nil
}
