package integrate

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/stretchr/testify/assert"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/headway/united"
	"github.com/itchio/mitch"
)

func Test_InstallCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cancel test in short mode")
	}
	assert := assert.New(t)

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

	store := bi.Server.Store()
	_developer := store.MakeUser("Ricky Machine")
	_game := _developer.MakeGame("Platformer Platitude")
	_game.Publish()
	_upload := _game.MakeUpload("tagged.zip")
	_upload.SetAllPlatforms()
	_upload.SetZipContentsCustom(func(ac *mitch.ArchiveContext) {
		ac.Entry("readme.txt").String("You can't play random binary data, silly face.")
		ac.Entry("random.bin").Random(0xfaceface, 16*1024*1024)
	})

	MiBPerSecond := int64(1024 * 8)
	_, err := messages.NetworkSetBandwidthThrottle.TestCall(rc, butlerd.NetworkSetBandwidthThrottleParams{
		Enabled: true,
		Rate:    4 * MiBPerSecond,
	})
	must(err)

	defer func() {
		_, err := messages.NetworkSetBandwidthThrottle.TestCall(rc, butlerd.NetworkSetBandwidthThrottleParams{
			Enabled: false,
		})
		must(err)
	}()

	game := bi.FetchGame(_game.ID)

	queueRes, err := messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
		Game:              game,
		InstallLocationID: "tmp",
	})
	must(err)

	pidFilePath := filepath.Join(queueRes.StagingFolder, "operate-pid.json")

	var lastProgressValue float64
	var lastProgressLock sync.Mutex

	printProgress := func(params butlerd.ProgressNotification) {
		lastProgressLock.Lock()
		defer lastProgressLock.Unlock()

		bi.Logf("%.2f%% done @ %s / s ETA %s", params.Progress*100, united.FormatBytes(int64(params.BPS)), time.Duration(params.ETA*float64(time.Second)))
		lastProgressValue = params.Progress
	}

	gracefulCancelOnce := &sync.Once{}

	messages.Progress.Register(h, func(params butlerd.ProgressNotification) {
		printProgress(params)

		if params.Progress > 0.2 {
			_, err := os.Stat(pidFilePath)
			assert.NoError(err, "pid file exists before we graceful cancel")

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

	lastProgressLock.Lock()
	bi.Logf("Last progress before graceful cancel: %.2f%%", lastProgressValue*100)
	lastProgressLock.Unlock()

	bi.Logf("Making sure we've been cancelled...")
	assert.Error(err)
	je := err.(*jsonrpc2.Error)
	assert.EqualValues(butlerd.CodeOperationCancelled, je.Code)

	bi.Logf("Resuming while offline...")
	offlineStart := time.Now()
	_, err = messages.NetworkSetSimulateOffline.TestCall(rc, butlerd.NetworkSetSimulateOfflineParams{
		Enabled: true,
	})
	must(err)
	bi.Logf("SetOffline took %s", time.Since(offlineStart))

	bi.Logf("Now calling installperform")
	_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
		ID:            queueRes.ID,
		StagingFolder: queueRes.StagingFolder,
	})
	assert.Error(err)
	je = err.(*jsonrpc2.Error)
	assert.EqualValues(butlerd.CodeNetworkDisconnected, je.Code)

	_, err = messages.NetworkSetSimulateOffline.TestCall(rc, butlerd.NetworkSetSimulateOfflineParams{
		Enabled: false,
	})
	must(err)

	bi.Logf("Resuming after graceful cancel...")

	hardCancelOnce := &sync.Once{}

	messages.Progress.Register(h, func(params butlerd.ProgressNotification) {
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

	lastProgressLock.Lock()
	bi.Logf("Last progress before hard cancel: %.2f%%", lastProgressValue*100)
	lastProgressLock.Unlock()
	assert.Error(err)

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
	assert.True(pidFileDisappeared, "pid file should disappear after cancellation (even hard)")
	bi.Logf("PID file disappeared in %s", time.Since(beforePidDisappear))

	bi.Logf("Resuming after hard cancel...")
	rc, h, _ = bi.Connect()

	messages.Progress.Register(h, func(params butlerd.ProgressNotification) {
		printProgress(params)
	})

	_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
		ID:            queueRes.ID,
		StagingFolder: queueRes.StagingFolder,
	})
	assert.NoError(err)
}
