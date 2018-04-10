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
	rc, h, cancel := connect(t)
	defer cancel()

	authenticate(t, rc)
	setupTmpInstallLocation(t, h, rc)

	{
		// fasterthanlime/butler
		game := getGame(t, h, rc, 239683)

		client := itchio.ClientWithKey(os.Getenv("ITCH_TEST_ACCOUNT_API_KEY"))
		res, err := client.GameUploads(game.ID)
		must(t, err)

		var upload *itchio.Upload
		for _, u := range res.Uploads {
			if u.ChannelName == "darwin-amd64-head" {
				upload = u
				break
			}
		}
		assert.NotNil(t, upload)

		queue1Res, err := messages.InstallQueue.TestCall(rc, &butlerd.InstallQueueParams{
			Game:              game,
			InstallLocationID: "tmp",
			Upload:            upload,
			Build: &itchio.Build{
				ID: 76706,
			},
		})
		must(t, err)

		caveId := queue1Res.CaveID
		assert.NotEmpty(t, caveId)

		_, err = messages.InstallPerform.TestCall(rc, &butlerd.InstallPerformParams{
			ID:            queue1Res.ID,
			StagingFolder: queue1Res.StagingFolder,
		})
		must(t, err)

		{
			_, err := os.Stat("./tmp/butler/.itch/receipt.json.gz")
			assert.NoError(t, err, "has receipt")
		}

		t.Logf("Upgrading to next build")

		queue2Res, err := messages.InstallQueue.TestCall(rc, &butlerd.InstallQueueParams{
			Game:              game,
			InstallLocationID: "tmp",
			CaveID:            caveId,
			Upload:            upload,
			Build: &itchio.Build{
				ID: 76741,
			},
		})
		must(t, err)

		assert.EqualValues(t, queue1Res.CaveID, queue2Res.CaveID, "installing for same cave")
		assert.EqualValues(t, queue1Res.InstallFolder, queue2Res.InstallFolder, "using same install folder")

		_, err = messages.InstallPerform.TestCall(rc, &butlerd.InstallPerformParams{
			ID:            queue2Res.ID,
			StagingFolder: queue2Res.StagingFolder,
		})
		must(t, err)
	}
}
