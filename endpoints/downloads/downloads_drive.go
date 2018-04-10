package downloads

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/itchio/httpkit/timeout"

	"github.com/itchio/httpkit/neterr"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/pkg/errors"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/cmd/wipe"
	"github.com/itchio/butler/database/models"
)

var downloadsDriveCancelID = "Downloads.Drive"

const pingURL = "https://itch.io/static/ping.txt"

type Status struct {
	Online bool
}

func DownloadsDrive(rc *butlerd.RequestContext, params *butlerd.DownloadsDriveParams) (*butlerd.DownloadsDriveResult, error) {
	consumer := rc.Consumer
	consumer.Infof("Now driving downloads...")

	parentCtx := rc.Ctx
	ctx, cancelFunc := context.WithCancel(parentCtx)

	rc.CancelFuncs.Add(downloadsDriveCancelID, cancelFunc)
	defer rc.CancelFuncs.Remove(downloadsDriveCancelID)

	status := &Status{
		Online: true,
	}

poll:
	for {
		select {
		case <-ctx.Done():
			consumer.Infof("Drive cancelled, bye!")
			break poll
		default:
			// let's keep going
		}

		err := cleanDiscarded(rc)
		if err != nil {
			consumer.Warnf("%+v", errors.WithMessage(err, "while cleaning discarded:"))
		}

		err = performOne(ctx, rc)
		if err != nil {
			if err == butlerd.CodeNetworkDisconnected {
				err = waitForInternet(rc, status)
				if err != nil {
					consumer.Warnf("%+v", errors.WithMessage(err, "while waiting for internet:"))
				}
			} else {
				consumer.Warnf("%+v", errors.WithMessage(err, "while performing download:"))
			}
		}

		time.Sleep(1 * time.Second)
	}

	res := &butlerd.DownloadsDriveResult{}
	return res, nil
}

func waitForInternet(rc *butlerd.RequestContext, status *Status) error {
	consumer := rc.Consumer

	// notify always, but only log once
	messages.DownloadsDriveNetworkStatus.Notify(rc, &butlerd.DownloadsDriveNetworkStatusNotification{
		Status: butlerd.NetworkStatusOffline,
	})
	if status.Online {
		status.Online = false
		consumer.Opf("Looks like we're offline! Waiting for an internet connection...")
	}

	client := timeout.NewDefaultClient()

	// wait up to 120 rounds (2 minutes if tries take 0s,
	// which they don't), then give up waiting
	for i := 0; i < 120; i++ {
		res, err := client.Get(pingURL)
		if err != nil {
			if neterr.IsNetworkError(err) {
				// keep going...
			}
		} else {
			payload, _ := ioutil.ReadAll(res.Body)
			consumer.Statf("Looks like we're back online! (%s)", strings.TrimSpace(string(payload)))
			messages.DownloadsDriveNetworkStatus.Notify(rc, &butlerd.DownloadsDriveNetworkStatusNotification{
				Status: butlerd.NetworkStatusOnline,
			})
			status.Online = true
			return nil
		}

		time.Sleep(1 * time.Second)
	}
	return nil
}

func cleanDiscarded(rc *butlerd.RequestContext) error {
	consumer := rc.Consumer

	var discardedDownloads []*models.Download
	err := rc.DB().Where(`discarded`).Find(&discardedDownloads).Error
	if err != nil {
		return errors.WithStack(err)
	}

	models.PreloadDownloads(rc.DB(), discardedDownloads)
	for _, download := range discardedDownloads {
		consumer.Opf("Cleaning up download for %s", operate.GameToString(download.Game))

		if download.StagingFolder == "" {
			consumer.Warnf("No staging folder specified, can't wipe")
		} else {
			consumer.Opf("Wiping staging folder...")
			err := wipe.Do(consumer, download.StagingFolder)
			if err != nil {
				consumer.Warnf("While wiping staging folder: %s", err.Error())
			}
		}

		if download.Fresh && (download.FinishedAt == nil || download.Error != nil) {
			if download.InstallFolder == "" {
				consumer.Warnf("No (fresh) install folder specified, can't wipe")
			} else {
				consumer.Opf("Wiping (fresh) install folder...")
				err := wipe.Do(consumer, download.InstallFolder)
				if err != nil {
					consumer.Warnf("While wiping (fresh) install folder: %s", err.Error())
				}
			}
		}

		err := rc.DB().Delete(download).Error
		if err != nil {
			return errors.WithStack(err)
		}

		messages.DownloadsDriveDiscarded.Notify(rc, &butlerd.DownloadsDriveDiscardedNotification{
			Download: formatDownload(download),
		})
	}
	return nil
}

func performOne(parentCtx context.Context, rc *butlerd.RequestContext) error {
	consumer := rc.Consumer

	var pendingDownloads []*models.Download
	err := rc.DB().Where(`finished_at IS NULL AND NOT discarded`).Order(`position ASC`).Find(&pendingDownloads).Error
	if err != nil {
		return errors.WithStack(err)
	}

	if len(pendingDownloads) == 0 {
		return nil
	}

	download := pendingDownloads[0]
	download.Preload(rc.DB())
	consumer.Infof("%d pending downloads, performing for %s", len(pendingDownloads), operate.GameToString(download.Game))

	ctx, cancelFunc := context.WithCancel(parentCtx)
	defer cancelFunc()

	wasDiscarded := func() bool {
		// have we been discarded?
		{
			var row = struct {
				Discarded bool
			}{}
			err := rc.DB().Raw(`SELECT discarded FROM downloads WHERE id = ?`, download.ID).Scan(&row).Error
			if err != nil {
				consumer.Warnf("Could not check whether download is discarded: %s", err.Error())
			}

			if row.Discarded {
				consumer.Infof("Download was cancelled from under us, bailing out!")
				return true
			}
		}

		// has something else been prioritized?
		{
			var row = struct {
				ID string
			}{}
			err := rc.DB().Raw(`SELECT id FROM downloads WHERE finished_at IS NULL AND NOT discarded ORDER BY position ASC LIMIT 1`).Scan(&row).Error
			if err != nil {
				consumer.Warnf("Could not check whether download is discarded: %s", err.Error())
			}

			if row.ID != download.ID {
				consumer.Infof("%s deprioritized (for %s), bailing out!", download.ID, row.ID)
				return true
			}
		}
		return false
	}
	goGadgetoDiscardWatcher := func() {
		for {
			select {
			case <-time.After(5 * time.Second):
				if wasDiscarded() {
					cancelFunc()
				}
			case <-ctx.Done():
				return
			}
		}
	}
	go goGadgetoDiscardWatcher()

	var stage = "prepare"
	var progress, eta, bps float64
	const maxSpeedDatapoints = 60
	speedHistory := make([]float64, maxSpeedDatapoints)

	lastProgress := time.Now()

	sendProgress := func() error {
		if time.Since(lastProgress).Seconds() < 0.5 {
			return nil
		}
		lastProgress = time.Now()

		speedHistory = append(speedHistory, bps)
		if len(speedHistory) > maxSpeedDatapoints {
			speedHistory = speedHistory[len(speedHistory)-maxSpeedDatapoints:]
		}

		return messages.DownloadsDriveProgress.Notify(rc, &butlerd.DownloadsDriveProgressNotification{
			Download: formatDownload(download),
			Progress: &butlerd.DownloadProgress{
				Stage:    stage,
				Progress: progress,
				ETA:      eta,
				BPS:      bps,
			},
			SpeedHistory: speedHistory,
		})
	}

	defer rc.StopInterceptingNotification(messages.Progress.Method())
	rc.InterceptNotification(messages.Progress.Method(), func(method string, paramsIn interface{}) error {
		params := paramsIn.(*butlerd.ProgressNotification)
		progress = params.Progress
		eta = params.ETA
		bps = params.BPS
		return sendProgress()
	})

	defer rc.StopInterceptingNotification(messages.TaskStarted.Method())
	rc.InterceptNotification(messages.TaskStarted.Method(), func(method string, paramsIn interface{}) error {
		params := paramsIn.(*butlerd.TaskStartedNotification)
		stage = string(params.Type)
		return sendProgress()
	})

	defer rc.StopInterceptingNotification(messages.TaskSucceeded.Method())
	rc.InterceptNotification(messages.TaskSucceeded.Method(), func(method string, paramsIn interface{}) error {
		return nil
	})

	err = func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				consumer.Warnf("Recovered from panic!")
				if rErr, ok := r.(error); ok {
					err = errors.WithStack(rErr)
				} else {
					err = errors.Errorf("%v", r)
				}
			}
		}()

		messages.DownloadsDriveStarted.Notify(rc, &butlerd.DownloadsDriveStartedNotification{
			Download: formatDownload(download),
		})

		err = operate.InstallPerform(ctx, rc, &butlerd.InstallPerformParams{
			ID:            download.ID,
			StagingFolder: download.StagingFolder,
		})
		return
	}()
	if err != nil {
		if wasDiscarded() {
			// download errored, but it was already discarded, ignoring.
			return nil
		}

		if be, ok := butlerd.AsButlerdError(err); ok {
			switch butlerd.Code(be.RpcErrorCode()) {
			case butlerd.CodeNetworkDisconnected:
				// propagate so we can wait for the connection to be re-established
				return butlerd.CodeNetworkDisconnected
			case butlerd.CodeOperationCancelled:
				// the whole drive was probably cancelled?
				return nil
			case butlerd.CodeOperationAborted:
				consumer.Warnf("Download aborted, cleaning it out.")
				err := rc.DB().Delete(download).Error
				if err != nil {
					return errors.WithStack(err)
				}
				return nil
			}

			code := be.RpcErrorCode()
			download.ErrorCode = &code
			msg := be.RpcErrorMessage()
			download.ErrorMessage = &msg
		} else {
			var code int64
			var msg string
			if neterr.IsNetworkError(err) {
				return butlerd.CodeNetworkDisconnected
			} else {
				code = int64(jsonrpc2.CodeInternalError)
				msg = err.Error()
			}
			download.ErrorCode = &code
			download.ErrorMessage = &msg
		}

		var errString = fmt.Sprintf("%+v", err)
		consumer.Warnf("Download errored: %s", errString)
		download.Error = &errString

		finishedAt := time.Now().UTC()
		download.FinishedAt = &finishedAt
		download.Save(rc.DB())

		messages.DownloadsDriveErrored.Notify(rc, &butlerd.DownloadsDriveErroredNotification{
			Download: formatDownload(download),
		})

		return nil
	}

	consumer.Infof("Download finished!")
	finishedAt := time.Now().UTC()
	download.FinishedAt = &finishedAt
	download.Save(rc.DB())

	messages.DownloadsDriveFinished.Notify(rc, &butlerd.DownloadsDriveFinishedNotification{
		Download: formatDownload(download),
	})

	return nil
}
