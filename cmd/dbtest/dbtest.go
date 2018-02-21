package dbtest

import (
	"log"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/database"
	"github.com/itchio/butler/database/models"
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
	db, err := database.Open(&database.OpenParams{})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	db.LogMode(true)

	allModels := []interface{}{
		&models.Profile{},
		&itchio.DownloadKey{},
		&itchio.Collection{},
		&models.CollectionGame{},
		&models.DashboardGame{},
		&itchio.Game{},
		&models.Download{},
		&models.Cave{},
	}

	err = db.AutoMigrate(allModels...).Error
	if err != nil {
		return errors.Wrap(err, 0)
	}

	for _, model := range allModels {
		dumpTable(db, model)
	}

	dl := &models.Download{
		ID:    "098gas-d098g-90asd8-089as0d9",
		Order: 1,
	}
	dl.SetGame(&itchio.Game{
		Title: "hello",
	})

	err = db.Save(dl).Error
	if err != nil {
		return errors.Wrap(err, 0)
	}

	var dl2 models.Download
	err = db.First(&dl2, "id = ?", dl.ID).Error
	if err != nil {
		return errors.Wrap(err, 0)
	}

	game, err := dl2.GetGame()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	log.Printf("Fetched game: %#v", game)

	var dl3 models.Download
	err = db.Order(`"order" desc`).First(&dl3).Error
	if err != nil {
		return errors.Wrap(err, 0)
	}

	log.Printf("dl3: %s", dl3)

	return nil
}

func dumpTable(db *gorm.DB, obj interface{}) {
	scope := db.NewScope(obj)
	dialect := scope.Dialect()

	ms := scope.GetModelStruct()
	log.Printf("table %s: %d fields", ms.TableName(db), len(ms.StructFields))
	for _, sf := range ms.StructFields {
		if sf.IsIgnored {
			continue
		}

		suffix := ""
		if sf.IsPrimaryKey {
			suffix = "(primary key)"
		}
		log.Printf("%s %v %s", sf.DBName, dialect.DataTypeOf(sf), suffix)
	}
}
