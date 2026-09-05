package integrate

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/stretchr/testify/assert"
)

// Stands in for partner.steam-api.com.
func fakePartnerServer(t *testing.T, goodKey string) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != goodKey {
			w.WriteHeader(403)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"applist":{"apps":{"app":[
			{"appid":480,"app_name":"Spacewar","app_type":"game"},
			{"appid":481,"app_name":"Spacewar Demo","app_type":"demo"}
		]}}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func Test_PublishSteamSync_PublisherKeyFlow(t *testing.T) {
	assert := assert.New(t)

	srv := fakePartnerServer(t, "GOODKEY")
	t.Setenv("BUTLER_STEAM_PARTNER_URL", srv.URL)

	bi := newInstance(t)
	rc, _, cancel := bi.Unwrap()
	defer cancel()

	status, err := messages.PublishSteamSyncGetStatus.TestCall(rc, butlerd.PublishSteamSyncGetStatusParams{})
	must(err)
	assert.False(status.LoggedIn)
	assert.False(status.HasPublisherKey)

	_, err = messages.PublishSteamSyncListApps.TestCall(rc, butlerd.PublishSteamSyncListAppsParams{})
	assert.Error(err)
	assert.EqualValues(butlerd.CodePublishSteamSyncNoPublisherKey, err.(*jsonrpc2.Error).Code)

	_, err = messages.PublishSteamSyncSetPublisherKey.TestCall(rc, butlerd.PublishSteamSyncSetPublisherKeyParams{Key: "BADKEY"})
	assert.Error(err)
	assert.EqualValues(butlerd.CodePublishSteamSyncPublisherKeyInvalid, err.(*jsonrpc2.Error).Code)

	setRes, err := messages.PublishSteamSyncSetPublisherKey.TestCall(rc, butlerd.PublishSteamSyncSetPublisherKeyParams{Key: "GOODKEY"})
	must(err)
	assert.EqualValues(2, setRes.AppCount)

	status, err = messages.PublishSteamSyncGetStatus.TestCall(rc, butlerd.PublishSteamSyncGetStatusParams{})
	must(err)
	assert.False(status.LoggedIn)
	assert.True(status.HasPublisherKey)

	apps, err := messages.PublishSteamSyncListApps.TestCall(rc, butlerd.PublishSteamSyncListAppsParams{})
	must(err)
	if assert.Len(apps.Apps, 2) {
		assert.EqualValues(480, apps.Apps[0].ID)
		assert.Equal("Spacewar", apps.Apps[0].Name)
		assert.Equal("demo", apps.Apps[1].Type)
	}

	_, err = messages.PublishSteamSyncRemovePublisherKey.TestCall(rc, butlerd.PublishSteamSyncRemovePublisherKeyParams{})
	must(err)
	status, err = messages.PublishSteamSyncGetStatus.TestCall(rc, butlerd.PublishSteamSyncGetStatusParams{})
	must(err)
	assert.False(status.HasPublisherKey)

	_, err = messages.PublishSteamSyncSetPublisherKey.TestCall(rc, butlerd.PublishSteamSyncSetPublisherKeyParams{Key: "GOODKEY"})
	must(err)
	_, err = messages.PublishSteamSyncLogout.TestCall(rc, butlerd.PublishSteamSyncLogoutParams{})
	must(err)
	status, err = messages.PublishSteamSyncGetStatus.TestCall(rc, butlerd.PublishSteamSyncGetStatusParams{})
	must(err)
	assert.False(status.LoggedIn)
	assert.False(status.HasPublisherKey)

	cancelRes, err := messages.PublishSteamSyncLoginCancel.TestCall(rc, butlerd.PublishSteamSyncLoginCancelParams{ID: "nope"})
	must(err)
	assert.False(cancelRes.DidCancel)
}
