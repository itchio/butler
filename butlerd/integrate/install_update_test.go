package integrate

import (
	"os"
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	itchio "github.com/itchio/go-itchio"
	"github.com/stretchr/testify/assert"
)

func Test_InstallUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()

	log := func(msg string) {
		bi.Logf("==========================")
		bi.Logf("= %s", msg)
		bi.Logf("==========================")
	}

	bi.Authenticate()

	{
		log("listing builds...")

		// fasterthanlime/butler
		game := getGame(t, h, rc, 239683)

		client := itchio.ClientWithKey(os.Getenv("ITCH_TEST_ACCOUNT_API_KEY"))
		res, err := client.ListGameUploads(itchio.ListGameUploadsParams{
			GameID: game.ID,
		})
		must(t, err)

		var upload *itchio.Upload
		for _, u := range res.Uploads {
			if u.ChannelName == "darwin-amd64-head" {
				upload = u
				break
			}
		}
		assert.NotNil(t, upload)

		buildsRes, err := client.ListUploadBuilds(itchio.ListUploadBuildsParams{
			UploadID: upload.ID,
		})
		must(t, err)

		recentBuild := buildsRes.Builds[1]
		olderBuild := buildsRes.Builds[4]

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

		{
			_, err := os.Stat("./tmp/butler/.itch/receipt.json.gz")
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
