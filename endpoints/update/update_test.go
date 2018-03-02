package update_test

import (
	"context"
	"testing"

	"github.com/mitchellh/mapstructure"

	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/mockharness"
	"github.com/itchio/butler/cmd/operate/loopbackconn"
	"github.com/itchio/butler/endpoints/update"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/wtest"
	"github.com/stretchr/testify/assert"
	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

func TestCheckUpdateMissingFields(t *testing.T) {
	wtest.Must(t, mockharness.With(func(harness buse.Harness) error {
		router := buse.NewRouter(nil, nil)
		update.Register(router)

		item := &buse.CheckUpdateItem{
			InstalledAt: "2017-04-04T09:32:00Z",
		}
		consumer := &state.Consumer{
			OnMessage: func(level string, message string) {
				t.Logf("[%s] [%s]", level, message)
			},
		}
		ctx := context.Background()
		conn := loopbackconn.New(consumer)

		checkUpdate := func(params *buse.CheckUpdateParams) (*buse.CheckUpdateResult, error) {
			rc := &buse.RequestContext{
				Ctx:            ctx,
				Conn:           conn,
				Consumer:       consumer,
				MansionContext: router.MansionContext,
				Harness:        harness,
			}
			return update.CheckUpdate(rc, params)
		}

		params := &buse.CheckUpdateParams{
			Items: []*buse.CheckUpdateItem{
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
			assert.Contains(t, res.Warnings[0], "missing credentials")
		}

		{
			item.Credentials = testCredentials
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
				DecodeHook:       mapstructure.StringToTimeHookFunc(itchio.APIDateFormat),
			})
			wtest.Must(t, err)
			wtest.Must(t, dec.Decode(testUpload()))

			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", httpmock.NewBytesResponder(404, nil))
			res, err := checkUpdate(params)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(res.Warnings))
			assert.Contains(t, res.Warnings[0], "Server returned 404")
		}

		{
			t.Logf("All uploads gone")
			httpmock.Reset()
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, map[string]interface{}{
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
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, map[string]interface{}{
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
			freshUpload["updated_at"] = "2018-01-01 04:12:00"
			otherUpload := testUpload()
			otherUpload["id"] = 235987
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, map[string]interface{}{
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
			freshUpload["updated_at"] = "2018-01-01 04:12:00"
			otherUpload := testUpload()
			otherUpload["id"] = 235987
			otherUpload["build"] = map[string]interface{}{
				"id": 65432,
			}
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, map[string]interface{}{
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
			freshUpload["updated_at"] = "2018-01-01 04:12:00"
			otherUpload := testUpload()
			otherUpload["id"] = 235987
			otherUpload["build"] = map[string]interface{}{
				"id": 65432,
			}
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, map[string]interface{}{
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
			freshUpload["updated_at"] = "2018-01-01 04:12:00"
			otherUpload := testUpload()
			otherUpload["id"] = 235987
			otherUpload["build"] = map[string]interface{}{
				"id": 65432,
			}
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, map[string]interface{}{
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
			freshUpload["updated_at"] = "2018-01-01 04:12:00"
			otherUpload := testUpload()
			otherUpload["id"] = 235987
			otherUpload["build"] = map[string]interface{}{
				"id": 65432,
			}
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, map[string]interface{}{
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

		{
			t.Logf("Same upload exactly, but installedAt is invalid")
			item.Build = nil
			item.InstalledAt = "some weird date format"
			httpmock.Reset()
			freshUpload := testUpload()
			otherUpload := testUpload()
			otherUpload["id"] = 235987
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, map[string]interface{}{
				"uploads": []interface{}{
					freshUpload,
					otherUpload,
				},
			}))
			res, err := checkUpdate(params)
			assert.NoError(t, err)
			assert.Equal(t, 0, len(res.Warnings))
			assert.Equal(t, 1, len(res.Updates))
		}

		return nil
	}))
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

var testCredentials = &buse.GameCredentials{
	DownloadKey: 0,
	APIKey:      "KEY",
}

func testUpload() map[string]interface{} {
	return map[string]interface{}{
		"id":         768,
		"filename":   "foobar.zip",
		"updated_at": "2017-02-03 12:13:00",
		"size":       6273984,
		"windows":    true,
		"type":       "default",
	}
}
