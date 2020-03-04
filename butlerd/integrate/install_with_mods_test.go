package integrate

import (
	"path/filepath"
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/hush"
	"github.com/itchio/mitch"
	"github.com/itchio/screw"
	"github.com/stretchr/testify/assert"
)

func Test_InstallWithMods(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, _, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

	store := bi.Server.Store()
	_developer := store.MakeUser("nox")
	_game := _developer.MakeGame("Very Moddable")
	_game.Publish()

	_upload := _game.MakeUpload("the-game")
	_upload.SetAllPlatforms()
	_upload.ChannelName = "main"

	_build1 := _upload.PushBuild(func(ac *mitch.ArchiveContext) {
		ac.SetName("the-game.zip")
		ac.Entry("game.exe").Random(0x1, 1*1024*1024)

		ac.Entry("level1").Chunks([]mitch.Chunk{
			{Seed: 0x10, Size: 512 * 1024},
			{Seed: 0x11, Size: 512 * 1024},
		})
		ac.Entry("level2").Random(0x20, 512*1024)
	})

	game := bi.FetchGame(_game.ID)

	bi.Logf("First install")
	installRes := bi.Install(butlerd.InstallQueueParams{
		Game: game,
	})

	caveRes, err := messages.FetchCave.TestCall(rc, butlerd.FetchCaveParams{
		CaveID: installRes.CaveID,
	})
	must(err)

	installFolder := caveRes.Cave.InstallInfo.InstallFolder

	bi.Logf("Modding level1 (changed between builds)")
	err = screw.WriteFile(filepath.Join(installFolder, "level1"), []byte("haha modded!"), 0644)
	must(err)

	_build2 := _upload.PushBuild(func(ac *mitch.ArchiveContext) {
		ac.SetName("the-game.zip")
		ac.Entry("game.exe").Random(0x1, 1*1024*1024)

		ac.Entry("level1").Chunks([]mitch.Chunk{
			{Seed: 0x10, Size: 512 * 1024},
			{Seed: 0x12, Size: 512 * 1024},
		})
		ac.Entry("level2").Random(0x20, 512*1024)
	})
	build2 := bi.FetchBuild(_build2.ID)

	upgradeRes := bi.Install(butlerd.InstallQueueParams{
		Game:   game,
		CaveID: installRes.CaveID,

		Build: build2,
	})

	{
		ev := bi.FindEvent(upgradeRes.Events, hush.InstallEventFallback)
		assert.EqualValues(ev.Fallback.Attempted, "upgrade")
		assert.EqualValues(ev.Fallback.NowTrying, "heal")
		assert.Contains(ev.Fallback.Problem.Error, "expected weak hash")
	}
	{
		ev := bi.FindEvent(upgradeRes.Events, hush.InstallEventHeal)
		assert.NotZero(ev.Heal.TotalCorrupted)
	}

	bi.Logf("Making sure build2 is correctly installed")
	bi.InstallAndVerify(butlerd.InstallQueueParams{
		Game:   game,
		CaveID: installRes.CaveID,
	})

	build1 := bi.FetchBuild(_build1.ID)
	bi.Logf("Reverting to build1")
	bi.Install(butlerd.InstallQueueParams{
		Game:   game,
		CaveID: installRes.CaveID,

		Build: build1,
	})
	bi.InstallAndVerify(butlerd.InstallQueueParams{
		Game:   game,
		CaveID: installRes.CaveID,
	})

	bi.Logf("Modding level2 (NOT changed between builds)")
	err = screw.WriteFile(filepath.Join(installFolder, "level2"), []byte("modded too lol"), 0644)
	must(err)

	bi.Logf("Upgrading again to build2")
	upgradeRes = bi.Install(butlerd.InstallQueueParams{
		Game:   game,
		CaveID: installRes.CaveID,

		Build: build2,
	})

	{
		assert.Empty(bi.FindEvents(upgradeRes.Events, hush.InstallEventFallback))
		assert.Empty(bi.FindEvents(upgradeRes.Events, hush.InstallEventHeal))
	}
}
