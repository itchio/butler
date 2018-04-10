package integrate

import (
	"fmt"
	"testing"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_DownloadsDrive(t *testing.T) {
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
			QueueDownload:     true,
		})
		must(t, err)

		t.Logf("Queued %s", queueRes.InstallFolder)

		_, err = messages.NetworkSetSimulateOffline.TestCall(rc, &butlerd.NetworkSetSimulateOfflineParams{
			Enabled: true,
		})
		must(t, err)

		var notifs []string

		messages.DownloadsDriveStarted.Register(h, func(rc *butlerd.RequestContext, params *butlerd.DownloadsDriveStartedNotification) {
			notifs = append(notifs, "started")
		})

		messages.DownloadsDriveNetworkStatus.Register(h, func(rc *butlerd.RequestContext, params *butlerd.DownloadsDriveNetworkStatusNotification) {
			notifs = append(notifs, fmt.Sprintf("network-%s", params.Status))

			if params.Status == butlerd.NetworkStatusOffline {
				_, err = messages.NetworkSetSimulateOffline.TestCall(rc, &butlerd.NetworkSetSimulateOfflineParams{
					Enabled: false,
				})
				must(t, err)
			}
		})

		driveDone := make(chan error)

		messages.DownloadsDriveErrored.Register(h, func(rc *butlerd.RequestContext, params *butlerd.DownloadsDriveErroredNotification) {
			notifs = append(notifs, fmt.Sprintf("errored"))
			t.Logf("Got errored:")
			dl := params.Download
			if dl.ErrorMessage != nil {
				t.Logf(" ==> %#v", *dl.ErrorMessage)
			}
			if dl.ErrorCode != nil {
				t.Logf(" ==> Code %d", *dl.ErrorCode)
			}
			t.Logf("Full notification log: %#v", notifs)
			driveDone <- errors.New("Got unexpected DriveErrored")
			t.FailNow()
		})

		messages.DownloadsDriveFinished.Register(h, func(rc *butlerd.RequestContext, params *butlerd.DownloadsDriveFinishedNotification) {
			notifs = append(notifs, fmt.Sprintf("finished"))

			_, err = messages.DownloadsDriveCancel.TestCall(rc, &butlerd.DownloadsDriveCancelParams{})
			must(t, err)
		})

		go func() {
			_, err = messages.DownloadsDrive.TestCall(rc, &butlerd.DownloadsDriveParams{})
			driveDone <- err
		}()

		select {
		case err := <-driveDone:
			t.Logf("Notifications received: %#v", notifs)
			assert.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Logf("Notifications received: %#v", notifs)
			must(t, errors.New("timed out!"))
		}
	}
}
