package integrate

import (
	"encoding/json"
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/mitch"
)

func Test_ChangeCase(t *testing.T) {
	bi := newInstance(t)
	rc, _, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

	dumpJSON := func(header string, payload interface{}) {
		bs, err := json.MarshalIndent(payload, "", "  ")
		must(err)
		bi.Logf("%s:\n%s", header, string(bs))
	}

	store := bi.Server.Store()
	_developer := store.MakeUser("John Doe")
	_game := _developer.MakeGame("Airplane Simulator")
	_game.Type = "html"
	_game.Publish()
	_upload := _game.MakeUpload("All platforms")

	_upload.SetAllPlatforms()
	_build1 := _upload.PushBuild(func(ac *mitch.ArchiveContext) {
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

	installRes, err := messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
		ID:            queueRes.ID,
		StagingFolder: queueRes.StagingFolder,
	})
	must(err)

	dumpJSON("Install events", installRes.Events)

	bi.Logf("Pushing second build...")

	_build2 := _upload.PushBuild(func(ac *mitch.ArchiveContext) {
		ac.Entry("index.html").String("<p>Hi!</p>")
		ac.Entry("Data/data1").Random(0x1, 1024)
		ac.Entry("Data/data2").Random(0x2, 1024)
		ac.Entry("Data/data3").Random(0x3, 1024)
	})

	bi.Logf("Now upgrading to second build...")

	caveID := queueRes.CaveID
	upload := bi.FetchUpload(_upload.ID)
	build1 := bi.FetchBuild(_build1.ID)
	build2 := bi.FetchBuild(_build2.ID)

	queueRes, err = messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
		Game: game,
		// make sure to install to same cave so that it ends up
		// being an upgrade and not a duplicate install
		CaveID:            caveID,
		InstallLocationID: "tmp",

		// force upgrade otherwise it's going to default
		// to reinstall
		Upload: upload,
		Build:  build2,
	})
	must(err)

	upgradeRes, err := messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
		ID:            queueRes.ID,
		StagingFolder: queueRes.StagingFolder,
	})
	must(err)

	dumpJSON("Upgrade events", upgradeRes.Events)

	bi.Logf("Now re-install (heal)")

	queueRes, err = messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
		Game:              game,
		CaveID:            caveID,
		InstallLocationID: "tmp",
	})
	must(err)

	reinstallRes, err := messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
		ID:            queueRes.ID,
		StagingFolder: queueRes.StagingFolder,
	})
	must(err)

	dumpJSON("Re-install events", reinstallRes.Events)

	bi.Logf("Revert to past build (heal)")

	queueRes, err = messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
		Game:              game,
		CaveID:            caveID,
		InstallLocationID: "tmp",

		Build: build1,
	})
	must(err)

	revertRes, err := messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
		ID:            queueRes.ID,
		StagingFolder: queueRes.StagingFolder,
	})
	must(err)

	dumpJSON("Revert events", revertRes.Events)
}
