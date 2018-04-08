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

func Test_HasOne(t *testing.T) {
	type Drawback struct {
		ID          int64
		Comment     string
		SpecialtyID string
	}

	type Specialty struct {
		ID        string
		CountryID int64
		Drawback  *Drawback
	}

	type Country struct {
		ID        int64
		Desc      string
		Specialty *Specialty
	}

	models := []interface{}{&Country{}, &Specialty{}, &Drawback{}}

	withContext(t, models, func(db *gorm.DB, c *hades.Context) {
		country := &Country{
			ID:   324,
			Desc: "Shmance",
			Specialty: &Specialty{
				ID: "complain",
				Drawback: &Drawback{
					ID:      1249,
					Comment: "bitterness",
				},
			},
		}
		assertCount := func(model interface{}, expectedCount int64) {
			t.Helper()
			var count int64
			wtest.Must(t, db.Model(model).Count(&count).Error)
			assert.EqualValues(t, expectedCount, count)
		}

		wtest.Must(t, c.Save(db, &hades.SaveParams{Record: country, Assocs: []string{"Specialty"}}))
		assertCount(&Country{}, 0)
		assertCount(&Specialty{}, 1)
		assertCount(&Drawback{}, 1)

		wtest.Must(t, c.Save(db, &hades.SaveParams{Record: country}))
		assertCount(&Country{}, 1)
		assertCount(&Specialty{}, 1)
		assertCount(&Drawback{}, 1)

		var countries []*Country
		for i := 0; i < 4; i++ {
			country := &Country{}
			wtest.Must(t, db.Find(&country, "id = ?", 324).Error)
			countries = append(countries, country)
		}

		wtest.Must(t, c.Preload(db, &hades.PreloadParams{
			Record: countries,
			Fields: []hades.PreloadField{
				{Name: "Specialty"},
				{Name: "Specialty.Drawback"},
			},
		}))
	})
}

func Test_HasMany(t *testing.T) {
	type Quality struct {
		ID           int64
		ProgrammerID int64
		Label        string
	}

	type Programmer struct {
		ID        int64
		Qualities []*Quality
	}

	models := []interface{}{&Quality{}, &Programmer{}}
	withContext(t, models, func(db *gorm.DB, c *hades.Context) {
		assertCount := func(model interface{}, expectedCount int64) {
			t.Helper()
			var count int64
			wtest.Must(t, db.Model(model).Count(&count).Error)
			assert.EqualValues(t, expectedCount, count)
		}

		p1 := &Programmer{
			ID: 3,
			Qualities: []*Quality{
				&Quality{ID: 9, Label: "Inspiration"},
				&Quality{ID: 10, Label: "Creativity"},
				&Quality{ID: 11, Label: "Ability to not repeat oneself"},
			},
		}
		wtest.Must(t, c.Save(db, &hades.SaveParams{Record: p1}))
		assertCount(&Programmer{}, 1)
		assertCount(&Quality{}, 3)

		p1.Qualities[2].Label = "Inspiration again"
		wtest.Must(t, c.Save(db, &hades.SaveParams{Record: p1}))
		assertCount(&Programmer{}, 1)
		assertCount(&Quality{}, 3)
		{
			q := &Quality{}
			wtest.Must(t, db.Find(q, "id = ?", 11).Error)
			assert.EqualValues(t, "Inspiration again", q.Label)
		}

		p2 := &Programmer{
			ID: 8,
			Qualities: []*Quality{
				&Quality{ID: 40, Label: "Peace"},
				&Quality{ID: 41, Label: "Serenity"},
			},
		}
		programmers := []*Programmer{p1, p2}
		wtest.Must(t, c.Save(db, &hades.SaveParams{Record: programmers}))
		assertCount(&Programmer{}, 2)
		assertCount(&Quality{}, 5)

		p1bis := &Programmer{ID: 3}
		pp := &hades.PreloadParams{
			Record: p1bis,
			Fields: []hades.PreloadField{
				hades.PreloadField{Name: "Qualities"},
			},
		}
		wtest.Must(t, c.Preload(db, pp))
		assert.EqualValues(t, 3, len(p1bis.Qualities), "preload has_many")

		wtest.Must(t, c.Preload(db, pp))
		assert.EqualValues(t, 3, len(p1bis.Qualities), "preload replaces, doesn't append")

		pp.Fields[0] = hades.PreloadField{
			Name:    "Qualities",
			OrderBy: "id asc",
		}
		wtest.Must(t, c.Preload(db, pp))
		assert.EqualValues(t, "Inspiration", p1bis.Qualities[0].Label, "orders by (asc)")

		pp.Fields[0] = hades.PreloadField{
			Name:    "Qualities",
			OrderBy: "id desc",
		}
		wtest.Must(t, c.Preload(db, pp))
		assert.EqualValues(t, "Inspiration again", p1bis.Qualities[0].Label, "orders by (desc)")

		// no fields
		assert.Error(t, c.Preload(db, &hades.PreloadParams{Record: p1bis}))

		// not a model
		assert.Error(t, c.Preload(db, &hades.PreloadParams{Record: 42, Fields: pp.Fields}))

		// non-existent relation
		assert.Error(t, c.Preload(db, &hades.PreloadParams{Record: p1bis, Fields: []hades.PreloadField{{Name: "Woops"}}}))
	})
}

type Language struct {
	ID    int64
	Words []*Word `gorm:"many2many:language_words"`
}

type Word struct {
	ID        string
	Comment   string
	Languages []*Language `gorm:"many2many:language_words"`
}

type LanguageWord struct {
	LanguageID int64  `gorm:"primary_key;auto_increment:false"`
	WordID     string `gorm:"primary_key;auto_increment:false"`
}

func Test_ManyToMany(t *testing.T) {
	models := []interface{}{&Language{}, &Word{}, &LanguageWord{}}
	withContext(t, models, func(db *gorm.DB, c *hades.Context) {
		fr := &Language{
			ID: 123,
			Words: []*Word{
				&Word{ID: "Plume"},
				&Word{ID: "Week-end"},
			},
		}
		t.Logf("saving just fr")
		wtest.Must(t, c.Save(db, &hades.SaveParams{
			Record: fr,
		}))

		assertCount := func(model interface{}, expectedCount int64) {
			t.Helper()
			var count int64
			wtest.Must(t, db.Model(model).Count(&count).Error)
			assert.EqualValues(t, expectedCount, count)
		}
		assertCount(&Language{}, 1)
		assertCount(&Word{}, 2)
		assertCount(&LanguageWord{}, 2)

		en := &Language{
			ID: 456,
			Words: []*Word{
				&Word{ID: "Plume"},
				&Word{ID: "Week-end"},
			},
		}
		t.Logf("saving fr+en")
		wtest.Must(t, c.Save(db, &hades.SaveParams{
			Record: []*Language{fr, en},
		}))

		assertCount(&Language{}, 2)
		assertCount(&Word{}, 2)
		assertCount(&LanguageWord{}, 4)

		t.Logf("saving partial joins ('add' words to english)")
		en.Words = []*Word{
			&Word{ID: "Wreck"},
			&Word{ID: "Nervous"},
		}
		wtest.Must(t, c.Save(db, &hades.SaveParams{
			Record:       []*Language{en},
			PartialJoins: []string{"LanguageWords"},
		}))

		assertCount(&Language{}, 2)
		assertCount(&Word{}, 4)
		assertCount(&LanguageWord{}, 6)

		t.Logf("replacing all english words")
		wtest.Must(t, c.Save(db, &hades.SaveParams{
			Record: []*Language{en},
		}))

		assertCount(&Language{}, 2)
		assertCount(&Word{}, 4)
		assertCount(&LanguageWord{}, 4)

		t.Logf("adding commentary")
		en.Words[0].Comment = "punk band reference"
		wtest.Must(t, c.Save(db, &hades.SaveParams{
			Record: []*Language{en},
		}))

		assertCount(&Language{}, 2)
		assertCount(&Word{}, 4)
		assertCount(&LanguageWord{}, 4)

		{
			w := &Word{}
			wtest.Must(t, db.Find(w, "id = ?", "Wreck").Error)
			assert.EqualValues(t, "punk band reference", w.Comment)
		}

		langs := []*Language{
			&Language{ID: fr.ID},
			&Language{ID: en.ID},
		}
		err := c.Preload(db, &hades.PreloadParams{
			Record: langs,
			Fields: []hades.PreloadField{
				{Name: "Words"},
			},
		})
		// many_to_many preload is not implemented
		assert.Error(t, err)
	})
}

type Profile struct {
	ID           int64
	ProfileGames []*ProfileGame
}

type Game struct {
	ID    int64
	Title string
}

type ProfileGame struct {
	ProfileID int64 `gorm:"primary_key;auto_increment:false"`
	Profile   *Profile

	GameID int64 `gorm:"primary_key;auto_increment:false"`
	Game   *Game

	Order int64
}

func Test_ManyToManyRevenge(t *testing.T) {
	models := []interface{}{&Profile{}, &ProfileGame{}, &Game{}}

	withContext(t, models, func(db *gorm.DB, c *hades.Context) {
		makeProfile := func() *Profile {
			return &Profile{
				ID: 389,
				ProfileGames: []*ProfileGame{
					&ProfileGame{
						Order: 1,
						Game: &Game{
							ID:    58372,
							Title: "First offensive",
						},
					},
					&ProfileGame{
						Order: 5,
						Game: &Game{
							ID:    235971,
							Title: "Seconds until midnight",
						},
					},
					&ProfileGame{
						Order: 7,
						Game: &Game{
							ID:    10598,
							Title: "Three was company",
						},
					},
				},
			}
		}
		p := makeProfile()
		c.Save(db, &hades.SaveParams{
			Record: p,
		})
	})
}

func Test_PreloadEdgeCases(t *testing.T) {
	type Bar struct {
		ID int64
	}

	type Foo struct {
		ID    int64
		BarID int64
		Bar   *Bar
	}

	models := []interface{}{&Foo{}, &Bar{}}

	withContext(t, models, func(db *gorm.DB, c *hades.Context) {
		// non-existent Bar
		f := &Foo{ID: 1, BarID: 999}
		wtest.Must(t, c.Preload(db, &hades.PreloadParams{
			Record: f,
			Fields: []hades.PreloadField{
				{Name: "Bar"},
			},
		}))

		// empty slice
		var foos []*Foo
		wtest.Must(t, c.Preload(db, &hades.PreloadParams{
			Record: foos,
			Fields: []hades.PreloadField{
				{Name: "Bar"},
			},
		}))
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
