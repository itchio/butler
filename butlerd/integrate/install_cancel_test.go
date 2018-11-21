package integrate

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"
	"github.com/stretchr/testify/assert"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/httpkit/progress"
)

func Test_InstallCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()

	authenticate(t, rc)

	_, err := messages.NetworkSetBandwidthThrottle.TestCall(rc, butlerd.NetworkSetBandwidthThrottleParams{
		Enabled: true,
		Rate:    32 * 1024,
	})
	must(t, err)

	defer func() {
		_, err := messages.NetworkSetBandwidthThrottle.TestCall(rc, butlerd.NetworkSetBandwidthThrottleParams{
			Enabled: false,
		})
		must(t, err)
	}()

	{
		// itch-test-account/big-assets
		game := getGame(t, h, rc, 243485)

		queueRes, err := messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
			Game:              game,
			InstallLocationID: "tmp",
		})
		must(t, err)

		pidFilePath := filepath.Join(queueRes.StagingFolder, "operate-pid.json")

		var lastProgressValue float64
		printProgress := func(params butlerd.ProgressNotification) {
			bi.Logf("%.2f%% done @ %s / s ETA %s", params.Progress*100, progress.FormatBytes(int64(params.BPS)), time.Duration(params.ETA*float64(time.Second)))
			lastProgressValue = params.Progress
		}

		gracefulCancelOnce := &sync.Once{}

		messages.Progress.Register(h, func(rc *butlerd.RequestContext, params butlerd.ProgressNotification) {
			printProgress(params)

			if params.Progress > 0.2 {
				_, err := os.Stat(pidFilePath)
				assert.NoError(t, err, "pid file exists before we graceful cancel")

				gracefulCancelOnce.Do(func() {
					delete(h.notificationHandlers, messages.Progress.Method())

					bi.Logf("Calling graceful cancel")
					messages.InstallCancel.TestCall(rc, butlerd.InstallCancelParams{
						ID: queueRes.ID,
					})
					bi.Logf("Graceful cancel called")
				})
			}
		})

		bi.Logf("Queued %s", queueRes.InstallFolder)

		_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
			ID:            queueRes.ID,
			StagingFolder: queueRes.StagingFolder,
		})

		bi.Logf("Last progress before graceful cancel: %.2f%%", lastProgressValue*100)
		bi.Logf("Making sure we've been cancelled...")
		assert.Error(t, err)
		je := err.(*jsonrpc2.Error)
		assert.EqualValues(t, butlerd.CodeOperationCancelled, je.Code)

		bi.Logf("Resuming while offline...")
		_, err = messages.NetworkSetSimulateOffline.TestCall(rc, butlerd.NetworkSetSimulateOfflineParams{
			Enabled: true,
		})
		must(t, err)

		_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
			ID:            queueRes.ID,
			StagingFolder: queueRes.StagingFolder,
		})
		assert.Error(t, err)
		je = err.(*jsonrpc2.Error)
		assert.EqualValues(t, butlerd.CodeNetworkDisconnected, je.Code)

		_, err = messages.NetworkSetSimulateOffline.TestCall(rc, butlerd.NetworkSetSimulateOfflineParams{
			Enabled: false,
		})
		must(t, err)

		bi.Logf("Resuming after graceful cancel...")

		hardCancelOnce := &sync.Once{}

		messages.Progress.Register(h, func(rc *butlerd.RequestContext, params butlerd.ProgressNotification) {
			printProgress(params)

			if params.Progress > 0.5 {
				hardCancelOnce.Do(func() {
					bi.Logf("Sending hard cancel")
					delete(h.notificationHandlers, messages.Progress.Method())
					bi.Logf("Disconnecting...")
					bi.Disconnect()
					bi.Logf("Okay, we disconnected")
				})
			}
		})

		_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
			ID:            queueRes.ID,
			StagingFolder: queueRes.StagingFolder,
		})

		bi.Logf("Last progress before hard cancel: %.2f%%", lastProgressValue*100)
		assert.Error(t, err)

		bi.Logf("Waiting for pid file to disappear...")
		pidFileDisappeared := false
		beforePidDisappear := time.Now()
		for i := 0; i < 100; i++ {
			_, err := os.Stat(pidFilePath)
			if err != nil && os.IsNotExist(err) {
				// good!
				pidFileDisappeared = true
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		assert.True(t, pidFileDisappeared, "pid file should disappear after cancellation (even hard)")
		bi.Logf("PID file disappeared in %s", time.Since(beforePidDisappear))

		bi.Logf("Resuming after hard cancel...")
		rc, h, _ = bi.Connect()

		messages.Progress.Register(h, func(rc *butlerd.RequestContext, params butlerd.ProgressNotification) {
			printProgress(params)
		})

		_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
			ID:            queueRes.ID,
			StagingFolder: queueRes.StagingFolder,
		})
		assert.NoError(t, err)
	}
}

func getGame(t *testing.T, h *handler, rc *butlerd.RequestContext, gameID int64) *itchio.Game {
	gameRes, err := messages.FetchGame.TestCall(rc, butlerd.FetchGameParams{
		GameID: gameID,
	})
	must(t, err)
	return gameRes.Game
}
