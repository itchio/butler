package dbtest

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/mansion"
	itchio "github.com/itchio/go-itchio"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("dbtest", "Run DB tests!")
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do())
}

func Do() error {
	appData := os.Getenv("APPDATA")
	dbPath := filepath.Join(appData, "itch", "marketdb", "local.db")

	db, err := gorm.Open("sqlite3", dbPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	db.LogMode(true)

	var game itchio.Game
	db.First(&game, "lower(title) like ?", "overland")

	js, err := json.MarshalIndent(game, "", "  ")
	if err != nil {
		return errors.Wrap(err, 0)
	}

	log.Printf("Game: %s", string(js))
	return nil
}
