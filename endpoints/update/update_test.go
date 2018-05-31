package update_test

import (
	"context"
	"testing"
	"time"

	"github.com/itchio/butler/database"
	"github.com/itchio/butler/database/models"
	"github.com/jinzhu/gorm"

	"github.com/mitchellh/mapstructure"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate/loopbackconn"
	"github.com/itchio/butler/endpoints/update"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/wtest"
	"github.com/stretchr/testify/assert"
	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

func TestCheckUpdateMissingFields(t *testing.T) {
	db, err := database.Open("file::memory:?cache=shared")
	wtest.Must(t, err)

	getClient := func(key string) *itchio.Client {
		c := itchio.ClientWithKey(key)
		httpmock.ActivateNonDefault(c.HTTPClient)
		return c
	}

	router := butlerd.NewRouter(db, getClient)
	update.Register(router)

	item := &butlerd.CheckUpdateItem{
		InstalledAt: time.Date(2017, 04, 04, 9, 32, 00, 0, time.UTC),
	}
	consumer := &state.Consumer{
		OnMessage: func(level string, message string) {
			t.Logf("[%s] [%s]", level, message)
		},
	}
	ctx := context.Background()
	conn := loopbackconn.New(consumer)

	err = database.Prepare(db)
	wtest.Must(t, err)

	var testCredentials = &models.Profile{
		ID:     1,
		APIKey: "KEY",
	}
	wtest.Must(t, db.Save(testCredentials).Error)

	checkUpdate := func(params *butlerd.CheckUpdateParams) (*butlerd.CheckUpdateResult, error) {
		rc := &butlerd.RequestContext{
			Ctx:      ctx,
			Conn:     conn,
			Consumer: consumer,
			DB:       func() *gorm.DB { return db },
			Client:   getClient,
		}
		return update.CheckUpdate(rc, params)
	}

	params := &butlerd.CheckUpdateParams{
		Items: []*butlerd.CheckUpdateItem{
			item,
		},
	}

	{
		res, err := checkUpdate(params)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(res.Warnings))
		assert.Contains(t, res.Warnings[0], "missing itemId")
	}

	{
		item.ItemID = "foo-bar"
		res, err := checkUpdate(params)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(res.Warnings))
		assert.Contains(t, res.Warnings[0], "missing game")
	}

	{
		item.Game = testGame
		res, err := checkUpdate(params)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(res.Warnings))
		assert.Contains(t, res.Warnings[0], "missing upload")
	}

	{
		dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			TagName:          "json",
			WeaklyTypedInput: true,
			Result:           &item.Upload,
		})
		wtest.Must(t, err)
		wtest.Must(t, dec.Decode(testUpload()))

		httpmock.RegisterResponder("GET", "https://api.itch.io/games/123/uploads", httpmock.NewBytesResponder(404, nil))
		res, err := checkUpdate(params)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(res.Warnings))
		assert.Contains(t, res.Warnings[0], "Server returned 404")
	}

	{
		t.Logf("All uploads gone")
		httpmock.Reset()
		httpmock.RegisterResponder("GET", "https://api.itch.io/games/123/uploads", mustJsonResponder(t, 200, map[string]interface{}{
			"uploads": nil,
		}))
		res, err := checkUpdate(params)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(res.Warnings))
		assert.Equal(t, 0, len(res.Updates))
	}

	{
		t.Logf("Same upload exactly")
		httpmock.Reset()
		freshUpload := testUpload()
		otherUpload := testUpload()
		otherUpload["id"] = 235987
		httpmock.RegisterResponder("GET", "https://api.itch.io/games/123/uploads", mustJsonResponder(t, 200, map[string]interface{}{
			"uploads": []interface{}{
				freshUpload,
				otherUpload,
			},
		}))
		res, err := checkUpdate(params)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(res.Warnings))
		assert.Equal(t, 0, len(res.Updates))
	}

	{
		t.Logf("Upload updated recently")
		httpmock.Reset()
		freshUpload := testUpload()
		freshUpload["updated_at"] = "2018-01-01T04:12:00Z"
		otherUpload := testUpload()
		otherUpload["id"] = 235987
		httpmock.RegisterResponder("GET", "https://api.itch.io/games/123/uploads", mustJsonResponder(t, 200, map[string]interface{}{
			"uploads": []interface{}{
				freshUpload,
				otherUpload,
			},
		}))
		res, err := checkUpdate(params)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(res.Warnings))
		assert.Equal(t, 1, len(res.Updates))
		assert.EqualValues(t, freshUpload["id"], res.Updates[0].Upload.ID)
	}

	{
		t.Logf("Upload went wharf")
		httpmock.Reset()
		freshUpload := testUpload()
		freshUpload["build"] = map[string]interface{}{
			"id": 1230,
		}
		freshUpload["updated_at"] = "2018-01-01T04:12:00Z"
		otherUpload := testUpload()
		otherUpload["id"] = 235987
		otherUpload["build"] = map[string]interface{}{
			"id": 65432,
		}
		httpmock.RegisterResponder("GET", "https://api.itch.io/games/123/uploads", mustJsonResponder(t, 200, map[string]interface{}{
			"uploads": []interface{}{
				freshUpload,
				otherUpload,
			},
		}))
		res, err := checkUpdate(params)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(res.Warnings))
		assert.Contains(t, res.Warnings[0], "have no build installed but fresh upload has one")
		assert.Equal(t, 0, len(res.Updates))
	}

	{
		item.Build = &itchio.Build{
			ID: 12345,
		}

		t.Logf("Same build (wharf)")
		httpmock.Reset()
		freshUpload := testUpload()
		freshUpload["build"] = map[string]interface{}{
			"id": item.Build.ID,
		}
		freshUpload["updated_at"] = "2018-01-01T04:12:00Z"
		otherUpload := testUpload()
		otherUpload["id"] = 235987
		otherUpload["build"] = map[string]interface{}{
			"id": 65432,
		}
		httpmock.RegisterResponder("GET", "https://api.itch.io/games/123/uploads", mustJsonResponder(t, 200, map[string]interface{}{
			"uploads": []interface{}{
				freshUpload,
				otherUpload,
			},
		}))
		res, err := checkUpdate(params)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(res.Warnings))
		assert.Equal(t, 0, len(res.Updates))
	}

	{
		item.Build = &itchio.Build{
			ID: 12345,
		}

		t.Logf("Greater build ID (wharf)")
		httpmock.Reset()
		freshUpload := testUpload()
		freshUpload["build"] = map[string]interface{}{
			"id": 12346,
		}
		freshUpload["updated_at"] = "2018-01-01T04:12:00Z"
		otherUpload := testUpload()
		otherUpload["id"] = 235987
		otherUpload["build"] = map[string]interface{}{
			"id": 65432,
		}
		httpmock.RegisterResponder("GET", "https://api.itch.io/games/123/uploads", mustJsonResponder(t, 200, map[string]interface{}{
			"uploads": []interface{}{
				freshUpload,
				otherUpload,
			},
		}))
		res, err := checkUpdate(params)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(res.Warnings))
		assert.Equal(t, 1, len(res.Updates))
		assert.EqualValues(t, item.ItemID, res.Updates[0].ItemID)
		assert.EqualValues(t, item.Game, res.Updates[0].Game)
		assert.EqualValues(t, freshUpload["id"], res.Updates[0].Upload.ID)
		assert.EqualValues(t, freshUpload["build"].(map[string]interface{})["id"], res.Updates[0].Build.ID)
	}

	{
		t.Logf("Upload went wharf-less")
		httpmock.Reset()
		freshUpload := testUpload()
		freshUpload["updated_at"] = "2018-01-01T04:12:00Z"
		otherUpload := testUpload()
		otherUpload["id"] = 235987
		otherUpload["build"] = map[string]interface{}{
			"id": 65432,
		}
		httpmock.RegisterResponder("GET", "https://api.itch.io/games/123/uploads", mustJsonResponder(t, 200, map[string]interface{}{
			"uploads": []interface{}{
				freshUpload,
				otherUpload,
			},
		}))
		res, err := checkUpdate(params)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(res.Warnings))
		assert.Contains(t, res.Warnings[0], "have a build installed but fresh upload has none")
		assert.Equal(t, 0, len(res.Updates))
	}
}

func mustJsonResponder(t *testing.T, status int, body interface{}) httpmock.Responder {
	r, err := httpmock.NewJsonResponder(status, body)
	wtest.Must(t, err)
	return r
}

var testGame = &itchio.Game{
	ID:    123,
	Title: "Not plausible",
	URL:   "https://insanity.itch.io/not-plausible",
}

func testUpload() map[string]interface{} {
	return map[string]interface{}{
		"id":         768,
		"filename":   "foobar.zip",
		"updated_at": "2017-02-03T12:13:00Z",
		"size":       6273984,
		"windows":    true,
		"type":       "default",
	}
}
