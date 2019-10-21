package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/mitch"
)

func Test_ChangeCase(t *testing.T) {
	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

	store := bi.Server.Store()
	_developer := store.MakeUser("John Doe")
	_game := _developer.MakeGame("Airplane Simulator")
	_game.Type = "html"
	_game.Publish()
	_upload := _game.MakeUpload("All platforms")
	_upload.SetAllPlatforms()
	_upload.SetZipContentsCustom(func(ac *mitch.ArchiveContext) {
		ac.Entry("index.html").String("<p>Hi!</p>")
		ac.Entry("data/data1").Random(0x1, 1024)
		ac.Entry("data/data2").Random(0x2, 1024)
		ac.Entry("data/data3").Random(0x3, 1024)
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
}
