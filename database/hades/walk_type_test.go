package hades_test

import (
	"testing"

	"github.com/itchio/butler/database/hades"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/wtest"
	"github.com/stretchr/testify/assert"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func Test_BelongsTo(t *testing.T) {
	type Fate struct {
		ID   int64
		Desc string
	}

	type Human struct {
		ID     int64
		FateID int64
		Fate   *Fate `gorm:"ignore"`
	}

	type Joke struct {
		ID      string
		HumanID int64
		Human   *Human `gorm:"ignore"`
	}

	models := []interface{}{&Human{}, &Fate{}, &Joke{}}

	withContext(t, models, func(db *gorm.DB, c *hades.Context) {
		someFate := &Fate{
			ID:   123,
			Desc: "Consumer-grade flamethrowers",
		}
		wtest.Must(t, db.Save(someFate).Error)

		lea := &Human{
			ID:     3,
			FateID: someFate.ID,
		}
		wtest.Must(t, db.Save(lea).Error)

		c.Preload(db, &hades.PreloadParams{
			Record: lea,
			Fields: []hades.PreloadField{
				{Name: "Fate"},
			},
		})
		assert.NotNil(t, lea.Fate)
		assert.EqualValues(t, someFate.Desc, lea.Fate.Desc)
	})

	withContext(t, models, func(db *gorm.DB, c *hades.Context) {
		lea := &Human{
			ID: 3,
			Fate: &Fate{
				ID:   421,
				Desc: "Book authorship",
			},
		}
		c.Save(db, &hades.SaveParams{
			Record: lea,
			Assocs: []string{"Fate"},
		})

		fate := &Fate{}
		wtest.Must(t, db.Where("id = ?", 421).Find(&fate).Error)
		assert.EqualValues(t, "Book authorship", fate.Desc)
	})

	withContext(t, models, func(db *gorm.DB, c *hades.Context) {
		fate := &Fate{
			ID:   3,
			Desc: "Space rodeo",
		}
		wtest.Must(t, db.Save(fate).Error)

		human := &Human{
			ID:     6,
			FateID: 3,
		}
		wtest.Must(t, db.Save(human).Error)

		joke := &Joke{
			ID:      "neuf",
			HumanID: 6,
		}
		wtest.Must(t, db.Save(joke).Error)

		c.Preload(db, &hades.PreloadParams{
			Record: joke,
			Fields: []hades.PreloadField{
				{Name: "Human"},
				{Name: "Human.Fate"},
			},
		})
		assert.NotNil(t, joke.Human)
		assert.NotNil(t, joke.Human.Fate)
		assert.EqualValues(t, "Space rodeo", joke.Human.Fate.Desc)
	})
}

func makeConsumer(t *testing.T) *state.Consumer {
	return &state.Consumer{
		OnMessage: func(lvl string, msg string) {
			t.Logf("[%s] %s", lvl, msg)
		},
	}
}

type testlogger struct {
	*testing.T
}

func (tl testlogger) Println(values ...interface{}) {
	tl.T.Log(values...)
}

type WithContextFunc func(db *gorm.DB, c *hades.Context)

func withContext(t *testing.T, models []interface{}, f WithContextFunc) {
	db, err := gorm.Open("sqlite3", ":memory:")
	wtest.Must(t, err)

	db.LogMode(true)
	db.SetLogger(gorm.Logger{testlogger{t}})
	defer db.Close()

	wtest.Must(t, db.AutoMigrate(models...).Error)

	c := hades.NewContext(db, models, makeConsumer(t))
	f(db, c)
}
