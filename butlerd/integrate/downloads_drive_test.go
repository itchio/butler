package integrate

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/itchio/mitch"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_DownloadsDrive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping downloads drive in short mode")
	}

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
	var notifsLock sync.Mutex
	appendNotif := func(s string) {
		notifsLock.Lock()
		notifs = append(notifs, s)
		notifsLock.Unlock()
	}

	messages.DownloadsDriveStarted.Register(h, func(params butlerd.DownloadsDriveStartedNotification) {
		appendNotif("started")
	})

	messages.DownloadsDriveNetworkStatus.Register(h, func(params butlerd.DownloadsDriveNetworkStatusNotification) {
		appendNotif(fmt.Sprintf("network-%s", params.Status))

		if params.Status == butlerd.NetworkStatusOffline {
			_, err := messages.NetworkSetSimulateOffline.TestCall(rc, butlerd.NetworkSetSimulateOfflineParams{
				Enabled: false,
			})
			must(err)
		}
	})

	driveDone := make(chan error)

	messages.DownloadsDriveErrored.Register(h, func(params butlerd.DownloadsDriveErroredNotification) {
		appendNotif(fmt.Sprintf("errored"))
		bi.Logf("Got errored:")
		dl := params.Download
		if dl.ErrorMessage != nil {
			bi.Logf(" ==> %#v", *dl.ErrorMessage)
		}
		if dl.ErrorCode != nil {
			bi.Logf(" ==> Code %d", *dl.ErrorCode)
		}
		notifsLock.Lock()
		bi.Logf("Full notification log: %#v", notifs)
		notifsLock.Unlock()
		driveDone <- errors.New("Got unexpected DriveErrored")
		t.FailNow()
	})

	messages.DownloadsDriveFinished.Register(h, func(params butlerd.DownloadsDriveFinishedNotification) {
		appendNotif(fmt.Sprintf("finished"))

		_, err := messages.DownloadsDriveCancel.TestCall(rc, butlerd.DownloadsDriveCancelParams{})
		must(err)
	})

	go func() {
		_, err := messages.DownloadsDrive.TestCall(rc, butlerd.DownloadsDriveParams{})
		driveDone <- err
	}()

	select {
	case err := <-driveDone:
		notifsLock.Lock()
		bi.Logf("Notifications received: %#v", notifs)
		notifsLock.Unlock()
		assert.NoError(err)
	case <-time.After(10 * time.Second):
		notifsLock.Lock()
		bi.Logf("Notifications received: %#v", notifs)
		notifsLock.Unlock()
		must(errors.New("timed out"))
	}

	_, err = messages.UninstallPerform.TestCall(rc, butlerd.UninstallPerformParams{
		CaveID: queueRes.CaveID,
	})
	must(err)
}
