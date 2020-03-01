package integrate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hush"
	"github.com/stretchr/testify/assert"
)

func (bi *ButlerInstance) SetupTmpInstallLocation() {
	wd, err := os.Getwd()
	must(err)

	tmpPath := filepath.Join(wd, "tmp")
	must(os.RemoveAll(tmpPath))
	must(os.MkdirAll(tmpPath, 0o755))

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

func (bi *ButlerInstance) FetchBuild(buildID int64) *itchio.Build {
	res, err := bi.Client().GetBuild(context.Background(), itchio.GetBuildParams{BuildID: buildID})
	must(err)
	return res.Build
}

func (bi *ButlerInstance) DumpJSON(header string, payload interface{}) {
	bs, err := json.MarshalIndent(payload, "", "  ")
	must(err)
	bi.Logf("%s:\n%s", header, string(bs))
}

func (bi *ButlerInstance) FindEvent(events []hush.InstallEvent, typ hush.InstallEventType) hush.InstallEvent {
	for _, ev := range events {
		if ev.Type == typ {
			return ev
		}
	}

	bi.DumpJSON(fmt.Sprintf("Needed to find %s in events", typ), events)
	panic(fmt.Sprintf("Could not find event of type %s", typ))
}

func (bi *ButlerInstance) FindEvents(events []hush.InstallEvent, typ hush.InstallEventType) []hush.InstallEvent {
	var results []hush.InstallEvent

	for _, ev := range events {
		if ev.Type == typ {
			results = append(results, ev)
		}
	}

	return results
}

func (bi *ButlerInstance) Install(params butlerd.InstallQueueParams) *butlerd.InstallPerformResult {
	rc := bi.Conn.RequestContext
	params.InstallLocationID = "tmp"

	queueRes, err := messages.InstallQueue.TestCall(rc, params)
	must(err)

	res, err := messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
		ID:            queueRes.ID,
		StagingFolder: queueRes.StagingFolder,
	})
	must(err)
	return res
}

func (bi *ButlerInstance) InstallAndVerify(params butlerd.InstallQueueParams) *butlerd.InstallPerformResult {
	assert := assert.New(bi.t)

	res := bi.Install(params)
	assert.Zero(bi.FindEvent(res.Events, hush.InstallEventHeal).Heal.TotalCorrupted)
	return res
}
