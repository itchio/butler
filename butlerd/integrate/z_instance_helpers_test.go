package integrate

import (
	"context"
	"os"
	"path/filepath"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	itchio "github.com/itchio/go-itchio"
	"github.com/stretchr/testify/assert"
)

func (bi *ButlerInstance) SetupTmpInstallLocation() {
	wd, err := os.Getwd()
	must(err)

	tmpPath := filepath.Join(wd, "tmp")
	must(os.RemoveAll(tmpPath))
	must(os.MkdirAll(tmpPath, 0755))

	rc := bi.Conn.RequestContext
	_, err = messages.InstallLocationsAdd.TestCall(rc, butlerd.InstallLocationsAddParams{
		ID:   "tmp",
		Path: filepath.Join(wd, "tmp"),
	})
	must(err)
}

const ConstantAPIKey = "butlerd integrate tests"

func (bi *ButlerInstance) Authenticate() *butlerd.Profile {
	store := bi.Server.Store()
	user := store.MakeUser("itch test account")
	apiKey := user.MakeAPIKey()
	apiKey.Key = ConstantAPIKey

	assert := assert.New(bi.t)

	rc := bi.Conn.RequestContext
	prof, err := messages.ProfileLoginWithAPIKey.TestCall(rc, butlerd.ProfileLoginWithAPIKeyParams{
		APIKey: apiKey.Key,
	})
	must(err)
	assert.EqualValues("itch test account", prof.Profile.User.DisplayName)

	return prof.Profile
}

// Client returns a go-itchio client configured against the mock server for
// this test.
func (bi *ButlerInstance) Client() *itchio.Client {
	client := itchio.ClientWithKey(ConstantAPIKey)
	client.SetServer("http://" + bi.Server.Address().String())
	return client
}

func (bi *ButlerInstance) FetchGame(gameID int64) *itchio.Game {
	rc := bi.Conn.RequestContext

	gameRes, err := messages.FetchGame.TestCall(rc, butlerd.FetchGameParams{
		GameID: gameID,
		Fresh:  true,
	})
	must(err)
	return gameRes.Game
}

func (bi *ButlerInstance) FetchUpload(uploadID int64) *itchio.Upload {
	res, err := bi.Client().GetUpload(context.Background(), itchio.GetUploadParams{UploadID: uploadID})
	must(err)
	return res.Upload
}
