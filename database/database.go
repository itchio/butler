package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/itchio/butler/database/models"
	"github.com/itchio/wharf/state"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	// enable sqlite3 dialect for gorm
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var debugSql = os.Getenv("BUTLER_SQL") == "1"

// Open returns a connection to butler's sqlite database
func Open(dbPath string) (*gorm.DB, error) {
	err := os.MkdirAll(filepath.Dir(dbPath), 0755)
	if err != nil {
		return nil, errors.Wrap(err, "creating db directory")
	}

	db, err := gorm.Open("sqlite3", dbPath+"?_busy_timeout=5000")
	if err != nil {
		return nil, errors.Wrap(err, "opening SQLite database")
	}

	if debugSql {
		db.LogMode(true)
	}

	// disable default gorm timestamp behavior, since our
	// "created_at/updated_at" fields typically come from
	// the API.
	db.Callback().Create().Remove("gorm:update_time_stamp")
	db.Callback().Update().Remove("gorm:update_time_stamp")

	return db, nil
}

// Prepare synchronizes schemas, runs migrations etc.
func Prepare(db *gorm.DB) error {
	err := db.AutoMigrate(models.AllModels...).Error
	if err != nil {
		return errors.WithMessage(err, "performing automatic DB migration")
	}

	return nil
}

// logging

func SetLogger(db *gorm.DB, consumer *state.Consumer) {
	db.SetLogger(&gorm.Logger{
		LogWriter: &consumerLogWriter{
			consumer: consumer,
		},
	})
}

type consumerLogWriter struct {
	consumer *state.Consumer
}

func (clw *consumerLogWriter) Println(args ...interface{}) {
	var tokens []string
	for _, arg := range args {
		tokens = append(tokens, fmt.Sprintf("%v", arg))
	}
	clw.consumer.Infof("%s", strings.Join(tokens, " "))
}
