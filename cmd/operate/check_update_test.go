package operate_test

import (
	"context"
	"testing"

	"gopkg.in/jarcoal/httpmock.v1"

	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/cmd/operate/harness"
	"github.com/itchio/butler/cmd/operate/harness/mockharness"
	"github.com/itchio/butler/cmd/operate/loopbackconn"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/wtest"
	"github.com/stretchr/testify/assert"
)

func TestCheckUpdateMissingFields(t *testing.T) {
	wtest.Must(t, mockharness.With(func(h harness.Harness) error {
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

		params := &buse.CheckUpdateParams{
			Items: []*buse.CheckUpdateItem{
				item,
			},
		}

		{
			res, err := operate.CheckUpdate(params, consumer, h, ctx, conn)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(res.Warnings))
			assert.Contains(t, res.Warnings[0], "missing itemId")
		}

		{
			item.ItemID = "foo-bar"
			res, err := operate.CheckUpdate(params, consumer, h, ctx, conn)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(res.Warnings))
			assert.Contains(t, res.Warnings[0], "missing credentials")
		}

		{
			item.Credentials = testCredentials
			res, err := operate.CheckUpdate(params, consumer, h, ctx, conn)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(res.Warnings))
			assert.Contains(t, res.Warnings[0], "missing game")
		}

		{
			item.Game = testGame
			res, err := operate.CheckUpdate(params, consumer, h, ctx, conn)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(res.Warnings))
			assert.Contains(t, res.Warnings[0], "missing upload")
		}

		{
			item.Upload = testUpload()
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", httpmock.NewBytesResponder(404, nil))
			res, err := operate.CheckUpdate(params, consumer, h, ctx, conn)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(res.Warnings))
			assert.Contains(t, res.Warnings[0], "Server returned 404")
		}

		{
			t.Logf("All uploads gone")
			httpmock.Reset()
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, &itchio.ListGameUploadsResponse{
				Uploads: nil,
			}))
			res, err := operate.CheckUpdate(params, consumer, h, ctx, conn)
			assert.NoError(t, err)
			assert.Equal(t, 0, len(res.Warnings))
			assert.Equal(t, 0, len(res.Updates))
		}

		{
			t.Logf("Same upload exactly")
			httpmock.Reset()
			freshUpload := testUpload()
			otherUpload := testUpload()
			otherUpload.ID = 235987
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, &itchio.ListGameUploadsResponse{
				Uploads: []*itchio.Upload{
					freshUpload,
					otherUpload,
				},
			}))
			res, err := operate.CheckUpdate(params, consumer, h, ctx, conn)
			assert.NoError(t, err)
			assert.Equal(t, 0, len(res.Warnings))
			assert.Equal(t, 0, len(res.Updates))
		}

		{
			t.Logf("Upload updated recently")
			httpmock.Reset()
			freshUpload := testUpload()
			freshUpload.UpdatedAt = "2018-01-01 04:12:00"
			otherUpload := testUpload()
			otherUpload.ID = 235987
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, &itchio.ListGameUploadsResponse{
				Uploads: []*itchio.Upload{
					freshUpload,
					otherUpload,
				},
			}))
			res, err := operate.CheckUpdate(params, consumer, h, ctx, conn)
			assert.NoError(t, err)
			assert.Equal(t, 0, len(res.Warnings))
			assert.Equal(t, 1, len(res.Updates))
			assert.EqualValues(t, freshUpload, res.Updates[0].Upload)
		}

		{
			t.Logf("Upload went wharf")
			httpmock.Reset()
			freshUpload := testUpload()
			freshUpload.Build = &itchio.Build{
				ID: 1230,
			}
			freshUpload.UpdatedAt = "2018-01-01 04:12:00"
			otherUpload := testUpload()
			otherUpload.ID = 235987
			otherUpload.Build = &itchio.Build{
				ID: 65432,
			}
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, &itchio.ListGameUploadsResponse{
				Uploads: []*itchio.Upload{
					freshUpload,
					otherUpload,
				},
			}))
			res, err := operate.CheckUpdate(params, consumer, h, ctx, conn)
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
			freshUpload.Build = item.Build
			freshUpload.UpdatedAt = "2018-01-01 04:12:00"
			otherUpload := testUpload()
			otherUpload.ID = 235987
			otherUpload.Build = &itchio.Build{
				ID: 65432,
			}
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, &itchio.ListGameUploadsResponse{
				Uploads: []*itchio.Upload{
					freshUpload,
					otherUpload,
				},
			}))
			res, err := operate.CheckUpdate(params, consumer, h, ctx, conn)
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
			freshUpload.Build = &itchio.Build{
				ID: 12346,
			}
			freshUpload.UpdatedAt = "2018-01-01 04:12:00"
			otherUpload := testUpload()
			otherUpload.ID = 235987
			otherUpload.Build = &itchio.Build{
				ID: 65432,
			}
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, &itchio.ListGameUploadsResponse{
				Uploads: []*itchio.Upload{
					freshUpload,
					otherUpload,
				},
			}))
			res, err := operate.CheckUpdate(params, consumer, h, ctx, conn)
			assert.NoError(t, err)
			assert.Equal(t, 0, len(res.Warnings))
			assert.Equal(t, 1, len(res.Updates))
			assert.EqualValues(t, item.ItemID, res.Updates[0].ItemID)
			assert.EqualValues(t, item.Game, res.Updates[0].Game)
			assert.EqualValues(t, freshUpload, res.Updates[0].Upload)
			assert.EqualValues(t, freshUpload.Build, res.Updates[0].Build)
		}

		{
			t.Logf("Upload went wharf-less")
			httpmock.Reset()
			freshUpload := testUpload()
			freshUpload.UpdatedAt = "2018-01-01 04:12:00"
			otherUpload := testUpload()
			otherUpload.ID = 235987
			otherUpload.Build = &itchio.Build{
				ID: 65432,
			}
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, &itchio.ListGameUploadsResponse{
				Uploads: []*itchio.Upload{
					freshUpload,
					otherUpload,
				},
			}))
			res, err := operate.CheckUpdate(params, consumer, h, ctx, conn)
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
			otherUpload.ID = 235987
			httpmock.RegisterResponder("GET", "https://itch.io/api/1/KEY/game/123/uploads", mustJsonResponder(t, 200, &itchio.ListGameUploadsResponse{
				Uploads: []*itchio.Upload{
					freshUpload,
					otherUpload,
				},
			}))
			res, err := operate.CheckUpdate(params, consumer, h, ctx, conn)
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

func testUpload() *itchio.Upload {
	return &itchio.Upload{
		ID:        768,
		Filename:  "foobar.zip",
		UpdatedAt: "2017-02-03 12:13:00",
		Size:      6273984,
		Windows:   true,
		Type:      "default",
	}
}
