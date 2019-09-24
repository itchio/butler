package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/mitch"
	"github.com/stretchr/testify/assert"
)

func Test_Configure(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

	store := bi.Server.Store()
	_developer := store.MakeUser("Amos 'Not my day' Wenger")
	_game := _developer.MakeGame("Sample HTML App")
	_game.Type = "html"
	_game.Publish()
	_upload := _game.MakeUpload("All platforms")
	_upload.SetAllPlatforms()
	_upload.SetZipContentsCustom(func(ac *mitch.ArchiveContext) {
		ac.Entry(".itch.toml").String(`
[[actions]]
name = "play"
path = "index.html"

[[actions]]
name = "manual"
path = "hello.txt"

[[actions]]
name = "forums"
path = "https://itch.io/community"
		`)
		ac.Entry("hello.txt").String("Just open index.html")
		ac.Entry("index.html").String("<html><body>Hello.</body></html>")
	})

	hadHTMLLaunch := false

	messages.HTMLLaunch.TestRegister(h, func(rc *butlerd.RequestContext, params butlerd.HTMLLaunchParams) (*butlerd.HTMLLaunchResult, error) {
		hadHTMLLaunch = true
		return &butlerd.HTMLLaunchResult{}, nil
	})

	game := bi.FetchGame(_game.ID)

	queueRes, err := messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
		Game:              game,
		InstallLocationID: "tmp",
	})
	must(err)

	bi.Logf("Queued %s", queueRes.InstallFolder)

	_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
		ID:            queueRes.ID,
		StagingFolder: queueRes.StagingFolder,
	})
	must(err)

	messages.PickManifestAction.TestRegister(h, func(rc *butlerd.RequestContext, params butlerd.PickManifestActionParams) (*butlerd.PickManifestActionResult, error) {
		bi.Logf("Got %d actions", len(params.Actions))
		for _, action := range params.Actions {
			bi.Logf("- %#v", action)
		}

		return &butlerd.PickManifestActionResult{
			Index: 0,
		}, nil
	})

	_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
		CaveID:     queueRes.CaveID,
		PrereqsDir: "/tmp/prereqs",
	})
	must(err)

	assert.True(hadHTMLLaunch)
}
