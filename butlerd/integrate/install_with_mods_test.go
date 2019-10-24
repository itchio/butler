package integrate

import (
	"path/filepath"
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/mitch"
	"github.com/itchio/screw"
)

func Test_InstallWithMods(t *testing.T) {
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

	_upload.PushBuild(func(ac *mitch.ArchiveContext) {
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

	bi.Logf("Modding some files...")

	caveRes, err := messages.FetchCave.TestCall(rc, butlerd.FetchCaveParams{
		CaveID: installRes.CaveID,
	})
	must(err)

	installFolder := caveRes.Cave.InstallInfo.InstallFolder

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

	bi.Install(butlerd.InstallQueueParams{
		Game:   game,
		CaveID: installRes.CaveID,

		Build: build2,
	})

	bi.InstallAndVerify(butlerd.InstallQueueParams{
		Game:   game,
		CaveID: installRes.CaveID,
	})
}
