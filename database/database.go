package database

import (
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type OpenParams struct {
	// defaults to "itch"
	AppName string
}

func Open(params *OpenParams) (*gorm.DB, error) {
	appName := params.AppName
	if appName == "" {
		appName = "itch"
	}

	dbPath, err := getDatabasePath(appName)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = os.MkdirAll(filepath.Dir(dbPath), 0755)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	db, err := gorm.Open("sqlite3", dbPath)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return db, nil
}

func getDatabasePath(appName string) (string, error) {
	appDataPath, err := GetAppDataPath(appName)
	if err != nil {
		return "", errors.Wrap(err, 0)
	}

	dbPath := filepath.Join(appDataPath, "db", "butler.db")
	return dbPath, nil
}
