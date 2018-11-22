package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/mitch"
)

func Test_InstallSmall(t *testing.T) {
	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

	store := bi.Server.Store()
	_developer := store.MakeUser("Roll Fizzlebeef")
	_game := _developer.MakeGame("Advent Burger Simulator")
	_game.Type = "html"
	_game.Publish()
	_upload := _game.MakeUpload("All platforms")
	_upload.SetAllPlatforms()
	_upload.SetZipContentsCustom(func(ac *mitch.ArchiveContext) {
		ac.Entry("index.html").String("<p>Hi!</p>")
		ac.Entry("styles/main.css").String("html { font-size: 16px; }")
		ac.Entry("data.pak").Random(0xfeedface, 4*1024*1024)
	})

	messages.HTMLLaunch.TestRegister(h, func(rc *butlerd.RequestContext, params butlerd.HTMLLaunchParams) (*butlerd.HTMLLaunchResult, error) {
		return &butlerd.HTMLLaunchResult{}, nil
	})

	{
		game := getGame(t, h, rc, _game.ID)

		queueRes, err := messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
			Game:              game,
			InstallLocationID: "tmp",
		})
		must(t, err)

		bi.Logf("Queued %s", queueRes.InstallFolder)

		_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
			ID:            queueRes.ID,
			StagingFolder: queueRes.StagingFolder,
		})
		must(t, err)

		_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
			CaveID:     queueRes.CaveID,
			PrereqsDir: "/tmp/prereqs",
		})
		must(t, err)

		_, err = messages.UninstallPerform.TestCall(rc, butlerd.UninstallPerformParams{
			CaveID: queueRes.CaveID,
		})
		must(t, err)
	}
}
