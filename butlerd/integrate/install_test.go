package integrate

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/stretchr/testify/assert"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	itchio "github.com/itchio/go-itchio"
)

func Test_InstallSmall(t *testing.T) {
	rc, h, cancel := connect(t)
	defer cancel()

	authenticate(t, rc)
	setupTmpInstallLocation(t, h, rc)

	{
		// itch-test-account/111-first
		game := getGame(t, h, rc, 149766)

		queueRes, err := messages.InstallQueue.TestCall(rc, &butlerd.InstallQueueParams{
			Game:              game,
			InstallLocationID: "tmp",
		})
		must(t, err)

		t.Logf("Queued %s", queueRes.InstallFolder)

		_, err = messages.InstallPerform.TestCall(rc, &butlerd.InstallPerformParams{
			ID:            queueRes.ID,
			StagingFolder: queueRes.StagingFolder,
		})
		must(t, err)

		messages.Launch.TestCall(rc, &butlerd.LaunchParams{
			CaveID: queueRes.CaveID,
		})
	}
}

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

func Test_InstallCancel(t *testing.T) {
	rc, h, cancel := connect(t)
	defer cancel()

	authenticate(t, rc)
	setupTmpInstallLocation(t, h, rc)

	{
		// itch-test-account/big-assets
		game := getGame(t, h, rc, 243485)

		queueRes, err := messages.InstallQueue.TestCall(rc, &butlerd.InstallQueueParams{
			Game:              game,
			InstallLocationID: "tmp",
		})
		must(t, err)

		var lastProgressValue float64
		printProgress := func(params *butlerd.ProgressNotification) {
			log.Printf("%.2f%% done @ %s / s ETA %s", params.Progress*100, humanize.IBytes(uint64(params.BPS)), time.Duration(params.ETA*float64(time.Second)))
			lastProgressValue = params.Progress
		}

		gracefulCancelOnce := &sync.Once{}

		messages.Progress.Register(h, func(rc *butlerd.RequestContext, params *butlerd.ProgressNotification) {
			printProgress(params)

			if params.Progress > 0.2 {
				gracefulCancelOnce.Do(func() {
					delete(h.notificationHandlers, messages.Progress.Method())

					messages.InstallCancel.TestCall(rc, &butlerd.InstallCancelParams{
						ID: queueRes.ID,
					})
				})
			}
		})

		t.Logf("Queued %s", queueRes.InstallFolder)

		_, err = messages.InstallPerform.TestCall(rc, &butlerd.InstallPerformParams{
			ID:            queueRes.ID,
			StagingFolder: queueRes.StagingFolder,
		})

		t.Logf("Last progress before graceful cancel: %.2f%%", lastProgressValue*100)
		t.Logf("Making sure we've been cancelled...")
		assert.Error(t, err)
		je := err.(*jsonrpc2.Error)
		assert.EqualValues(t, butlerd.CodeOperationCancelled, je.Code)

		t.Logf("Resuming while offline...")
		_, err = messages.NetworkSetSimulateOffline.TestCall(rc, &butlerd.NetworkSetSimulateOfflineParams{
			Enabled: true,
		})
		must(t, err)

		_, err = messages.InstallPerform.TestCall(rc, &butlerd.InstallPerformParams{
			ID:            queueRes.ID,
			StagingFolder: queueRes.StagingFolder,
		})
		assert.Error(t, err)
		je = err.(*jsonrpc2.Error)
		assert.EqualValues(t, butlerd.CodeNetworkDisconnected, je.Code)

		_, err = messages.NetworkSetSimulateOffline.TestCall(rc, &butlerd.NetworkSetSimulateOfflineParams{
			Enabled: false,
		})
		must(t, err)

		t.Logf("Resuming after graceful cancel...")

		messages.Progress.Register(h, func(rc *butlerd.RequestContext, params *butlerd.ProgressNotification) {
			printProgress(params)
		})

		hardCancelOnce := &sync.Once{}

		messages.Progress.Register(h, func(rc *butlerd.RequestContext, params *butlerd.ProgressNotification) {
			printProgress(params)

			if params.Progress > 0.5 {
				hardCancelOnce.Do(func() {
					delete(h.notificationHandlers, messages.Progress.Method())
					cancel()
				})
			}
		})

		_, err = messages.InstallPerform.TestCall(rc, &butlerd.InstallPerformParams{
			ID:            queueRes.ID,
			StagingFolder: queueRes.StagingFolder,
		})

		t.Logf("Last progress before hard cancel: %.2f%%", lastProgressValue*100)
		assert.Error(t, err)

		t.Logf("Resuming after hard cancel...")
		rc, h, cancel = connect(t)

		messages.Progress.Register(h, func(rc *butlerd.RequestContext, params *butlerd.ProgressNotification) {
			printProgress(params)
		})

		_, err = messages.InstallPerform.TestCall(rc, &butlerd.InstallPerformParams{
			ID:            queueRes.ID,
			StagingFolder: queueRes.StagingFolder,
		})
		assert.NoError(t, err)
	}
}

func setupTmpInstallLocation(t *testing.T, h *handler, rc *butlerd.RequestContext) {
	wd, err := os.Getwd()
	must(t, err)

	tmpPath := filepath.Join(wd, "tmp")
	must(t, os.RemoveAll(tmpPath))

	_, err = messages.InstallLocationsAdd.TestCall(rc, &butlerd.InstallLocationsAddParams{
		ID:   "tmp",
		Path: filepath.Join(wd, "tmp"),
	})
	must(t, err)
}

func getGame(t *testing.T, h *handler, rc *butlerd.RequestContext, gameID int64) *itchio.Game {
	gameChan := make(chan *itchio.Game)
	once := &sync.Once{}

	messages.FetchGameYield.Register(h, func(rc *butlerd.RequestContext, params *butlerd.FetchGameYieldNotification) {
		once.Do(func() {
			gameChan <- params.Game
		})
	})

	_, err := messages.FetchGame.TestCall(rc, &butlerd.FetchGameParams{
		GameID: gameID,
	})
	must(t, err)
	return <-gameChan
}
