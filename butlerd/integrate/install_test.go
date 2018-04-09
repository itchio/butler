package integrate

import (
	"log"
	"os"
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

func Test_Install(t *testing.T) {
	rc, h, cancel := connect(t)
	defer cancel()

	authenticate(t, rc)

	getGame := func(gameID int64) *itchio.Game {
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

	{
		t.Logf("Setting up install location")
		must(t, os.RemoveAll("./tmp"))

		_, err := messages.InstallLocationsAdd.TestCall(rc, &butlerd.InstallLocationsAddParams{
			ID:   "tmp",
			Path: "./tmp",
		})
		must(t, err)
	}

	{
		t.Logf("Installing something small...")

		// itch-test-account/111-first
		game := getGame(149766)

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

	{
		t.Logf("Installing something larger...")

		// itch-test-account/big-assets
		game := getGame(243485)

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
