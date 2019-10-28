package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/mitch"
	"github.com/itchio/screw"
	"github.com/stretchr/testify/assert"
)

func Test_ChangeCase(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	_, _, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

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

	installRes := bi.Install(butlerd.InstallQueueParams{
		Game: game,
	})
	assert.NotEmpty(installRes.CaveID)

	bi.Logf("Pushing second build...")

	_build2 := _upload.PushBuild(func(ac *mitch.ArchiveContext) {
		ac.Entry("index.html").String("<p>Hi!</p>")
		ac.Entry("Data/data1").Random(0x1, 1024)
		ac.Entry("Data/data2").Random(0x2, 1024)
		ac.Entry("Data/data3").Random(0x3, 1024)
	})

	bi.Logf("Now upgrading to second build...")

	upload := bi.FetchUpload(_upload.ID)
	build1 := bi.FetchBuild(_build1.ID)
	build2 := bi.FetchBuild(_build2.ID)

	upgradeRes := bi.Install(butlerd.InstallQueueParams{
		Game:   game,
		CaveID: installRes.CaveID,

		Upload: upload,
		Build:  build2,
	})
	bi.FindEvent(upgradeRes.Events, butlerd.InstallEventUpgrade)
	assert.Len(bi.FindEvents(upgradeRes.Events, butlerd.InstallEventPatching), 1)

	bi.InstallAndVerify(butlerd.InstallQueueParams{
		Game:   game,
		CaveID: installRes.CaveID,
	})

	bi.Logf("Revert to past build (heal)")
	revertRes := bi.Install(butlerd.InstallQueueParams{
		Game:   game,
		CaveID: installRes.CaveID,

		Build: build1,
	})

	{
		ev := bi.FindEvent(revertRes.Events, butlerd.InstallEventHeal)
		if screw.IsCaseInsensitiveFS() {
			assert.True(ev.Heal.AppliedCaseFixes)
			assert.Zero(ev.Heal.TotalCorrupted)
		} else {
			assert.False(ev.Heal.AppliedCaseFixes)
			assert.NotZero(ev.Heal.TotalCorrupted)
		}
	}

	bi.Logf("Making sure revert actually went fine")
	bi.InstallAndVerify(butlerd.InstallQueueParams{
		Game:   game,
		CaveID: installRes.CaveID,
	})
}
