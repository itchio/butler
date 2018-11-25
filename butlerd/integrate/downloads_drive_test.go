package integrate

import (
	"fmt"
	"testing"
	"time"

	"github.com/itchio/mitch"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_DownloadsDrive(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

	store := bi.Server.Store()
	_developer := store.MakeUser("Adam's Atom, eek")
	_game := _developer.MakeGame("Some web game")
	_game.Publish()
	_upload := _game.MakeUpload("web version")
	_upload.SetAllPlatforms()
	_upload.PushBuild(func(ac *mitch.ArchiveContext) {
		ac.SetName("html5.zip")
		ac.Entry("index.html").String("<p>Well hello</p>")
	})

	game := bi.FetchGame(_game.ID)

	queueRes, err := messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
		Game:              game,
		InstallLocationID: "tmp",
		QueueDownload:     true,
	})
	must(err)

	bi.Logf("Queued %s", queueRes.InstallFolder)

	_, err = messages.NetworkSetSimulateOffline.TestCall(rc, butlerd.NetworkSetSimulateOfflineParams{
		Enabled: true,
	})
	must(err)

	var notifs []string

	messages.DownloadsDriveStarted.Register(h, func(rc *butlerd.RequestContext, params butlerd.DownloadsDriveStartedNotification) {
		notifs = append(notifs, "started")
	})

	messages.DownloadsDriveNetworkStatus.Register(h, func(rc *butlerd.RequestContext, params butlerd.DownloadsDriveNetworkStatusNotification) {
		notifs = append(notifs, fmt.Sprintf("network-%s", params.Status))

		if params.Status == butlerd.NetworkStatusOffline {
			_, err = messages.NetworkSetSimulateOffline.TestCall(rc, butlerd.NetworkSetSimulateOfflineParams{
				Enabled: false,
			})
			must(err)
		}
	})

	driveDone := make(chan error)

	messages.DownloadsDriveErrored.Register(h, func(rc *butlerd.RequestContext, params butlerd.DownloadsDriveErroredNotification) {
		notifs = append(notifs, fmt.Sprintf("errored"))
		bi.Logf("Got errored:")
		dl := params.Download
		if dl.ErrorMessage != nil {
			bi.Logf(" ==> %#v", *dl.ErrorMessage)
		}
		if dl.ErrorCode != nil {
			bi.Logf(" ==> Code %d", *dl.ErrorCode)
		}
		bi.Logf("Full notification log: %#v", notifs)
		driveDone <- errors.New("Got unexpected DriveErrored")
		t.FailNow()
	})

	messages.DownloadsDriveFinished.Register(h, func(rc *butlerd.RequestContext, params butlerd.DownloadsDriveFinishedNotification) {
		notifs = append(notifs, fmt.Sprintf("finished"))

		_, err = messages.DownloadsDriveCancel.TestCall(rc, butlerd.DownloadsDriveCancelParams{})
		must(err)
	})

	go func() {
		_, err = messages.DownloadsDrive.TestCall(rc, butlerd.DownloadsDriveParams{})
		driveDone <- err
	}()

	select {
	case err := <-driveDone:
		bi.Logf("Notifications received: %#v", notifs)
		assert.NoError(err)
	case <-time.After(10 * time.Second):
		bi.Logf("Notifications received: %#v", notifs)
		must(errors.New("timed out"))
	}

	_, err = messages.UninstallPerform.TestCall(rc, butlerd.UninstallPerformParams{
		CaveID: queueRes.CaveID,
	})
	must(err)
}
