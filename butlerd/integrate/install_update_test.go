package integrate

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/mitch"
	"github.com/stretchr/testify/assert"
)

func Test_InstallUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()

	log := func(format string, args ...interface{}) {
		bi.Logf("==========================")
		bi.Logf("= %s", fmt.Sprintf(format, args...))
		bi.Logf("==========================")
	}

	bi.Authenticate()

	log("Simulate pushing builds...")

	store := bi.Server.Store()
	_developer := store.MakeUser("Zapp Brannigan")
	_game := _developer.MakeGame("Most erratic")
	_game.Publish()
	_upload := _game.MakeUpload("web version")
	_upload.SetAllPlatforms()
	_upload.ChannelName = "html5-head"
	constantSeed := int64(0x05adface)
	_upload.PushBuild(func(ac *mitch.ArchiveContext) {
		ac.SetName("html5.zip")
		ac.Entry("index.html").String("<p>This is version 1</p>")
		ac.Entry("data1.bin").Random(constantSeed, 4*1024*1024)
		ac.Entry("data2.bin").Random(0x0badface, 1*1024*1024)
	})
	_upload.PushBuild(func(ac *mitch.ArchiveContext) {
		ac.SetName("html5.zip")
		ac.Entry("index.html").String("<p>This is version 2</p>")
		ac.Entry("data1.bin").Random(constantSeed, 4*1024*1024)
		ac.Entry("data2.bin").Random(0x0dabface, 1*1024*1024)
	})
	_upload.PushBuild(func(ac *mitch.ArchiveContext) {
		ac.SetName("html5.zip")
		ac.Entry("index.html").String("<p>This is version 3</p>")
		ac.Entry("data1.bin").Random(constantSeed, 4*1024*1024)
		ac.Entry("data2.bin").Random(0x0dabfaa, 1*1024*1024)
	})
	_upload.PushBuild(func(ac *mitch.ArchiveContext) {
		ac.SetName("html5.zip")
		ac.Entry("index.html").String("<p>This is version 3</p>")
		ac.Entry("data1.bin").Random(0xfaaaaaa, 1*1024*1024)
		ac.Entry("data2.bin").Random(constantSeed, 2*1024*1024)
	})

	{
		log("listing uploads")

		game := getGame(t, h, rc, _game.ID)

		client := itchio.ClientWithKey(ConstantAPIKey)
		client.SetServer("http://" + bi.Server.Address().String())
		res, err := client.ListGameUploads(itchio.ListGameUploadsParams{
			GameID: game.ID,
		})
		must(t, err)

		log("got %d uploads", len(res.Uploads))

		var upload *itchio.Upload
		for _, u := range res.Uploads {
			if u.ChannelName == "html5-head" {
				upload = u
				break
			}
		}
		assert.NotNil(t, upload)

		buildsRes, err := client.ListUploadBuilds(itchio.ListUploadBuildsParams{
			UploadID: upload.ID,
		})
		must(t, err)

		log("got %d builds", len(buildsRes.Builds))

		recentBuild := buildsRes.Builds[0]
		olderBuild := buildsRes.Builds[2]

		log("installing older build...")

		queue1Res, err := messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
			Game:              game,
			InstallLocationID: "tmp",
			Upload:            upload,
			Build:             olderBuild,
		})
		must(t, err)

		caveID := queue1Res.CaveID
		assert.NotEmpty(t, caveID)

		_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
			ID:            queue1Res.ID,
			StagingFolder: queue1Res.StagingFolder,
		})
		must(t, err)

		caveRes, err := messages.FetchCave.TestCall(rc, butlerd.FetchCaveParams{
			CaveID: caveID,
		})
		must(t, err)
		cave := caveRes.Cave

		{
			_, err := os.Stat(filepath.Join(cave.InstallInfo.InstallFolder, ".itch/receipt.json.gz"))
			assert.NoError(t, err, "has receipt")
		}

		log("upgrading to next build...")

		queue2Res, err := messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
			Game:              game,
			InstallLocationID: "tmp",
			CaveID:            caveID,
			Upload:            upload,
			Build:             recentBuild,
		})
		must(t, err)

		assert.EqualValues(t, queue1Res.CaveID, queue2Res.CaveID, "installing for same cave")
		assert.EqualValues(t, queue1Res.InstallFolder, queue2Res.InstallFolder, "using same install folder")

		_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
			ID:            queue2Res.ID,
			StagingFolder: queue2Res.StagingFolder,
		})
		must(t, err)
	}
}
