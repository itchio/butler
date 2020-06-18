package integrate

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/itchio/butler/cmd/verify"

	"github.com/itchio/butler/cmd/sign"
	"github.com/itchio/wharf/pwr"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/mitch"
	"github.com/stretchr/testify/assert"
)

func Test_InstallUpdateNonWharf(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, _, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

	store := bi.Server.Store()
	_developer := store.MakeUser("Fry")
	_game := _developer.MakeGame("Omegaphone")
	_game.Publish()

	_up1 := _game.MakeUpload("v1.0.0")
	_up1.SetAllPlatforms()
	_up1.SetZipContentsCustom(func(ac *mitch.ArchiveContext) {
		ac.Entry("readme.txt").String("Not actually a game")
		ac.Entry("data/file1.xml").String("a long file 1 yep sir")
		ac.Entry("data/file2.xml").String("file 2 here")
		ac.Entry("data2/file12.xml").String("file 12 here")
	})

	makeUp2Contents := func(ac *mitch.ArchiveContext) {
		ac.Entry("readme.txt").String("Not actually a game")
		ac.Entry("data/file1.xml").String("shorter woop")
		ac.Entry("data/file3.xml").String("file 3 here")
	}

	_up2 := _game.MakeUpload("v2.0.0")
	_up2.SetAllPlatforms()
	_up2.SetZipContentsCustom(makeUp2Contents)

	_up2Bis := _game.MakeUpload("v2.0.0-bis")
	_up2Bis.SetAllPlatforms()
	_up2Bis.SetZipContentsCustom(makeUp2Contents)

	game := bi.FetchGame(_game.ID)

	var caveID string

	for _, _up := range []*mitch.Upload{_up1, _up2} {
		up := bi.FetchUpload(_up.ID)
		queueRes, err := messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
			Game:              game,
			Upload:            up,
			CaveID:            caveID,
			InstallLocationID: "tmp",
		})
		must(err)

		caveID = queueRes.CaveID

		_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
			ID:            queueRes.ID,
			StagingFolder: queueRes.StagingFolder,
		})
		must(err)
	}

	var referenceCaveID string

	{
		up := bi.FetchUpload(_up2Bis.ID)
		queueRes, err := messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
			Game:              game,
			Upload:            up,
			InstallLocationID: "tmp",
		})
		must(err)

		referenceCaveID = queueRes.CaveID

		_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
			ID:            queueRes.ID,
			StagingFolder: queueRes.StagingFolder,
		})
		must(err)
	}

	var actualFolder string
	var referenceFolder string

	{
		caveRes, err := messages.FetchCave.TestCall(rc, butlerd.FetchCaveParams{
			CaveID: caveID,
		})
		must(err)
		actualFolder = caveRes.Cave.InstallInfo.InstallFolder
	}

	{
		caveRes, err := messages.FetchCave.TestCall(rc, butlerd.FetchCaveParams{
			CaveID: referenceCaveID,
		})
		must(err)
		referenceFolder = caveRes.Cave.InstallInfo.InstallFolder
	}

	tmpDir, err := ioutil.TempDir("", "nonwharf-test")
	must(err)

	must(os.MkdirAll(tmpDir, 0o755))

	sigPath := filepath.Join(tmpDir, "expected-sig.pws")
	err = sign.Do(referenceFolder, sigPath, pwr.CompressionSettings{}, false)
	must(err)

	err = verify.Do(verify.Args{
		SignaturePath: sigPath,
		Dir:           actualFolder,
	})
	assert.NoError(err)
}
