package downloads

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/itchio/wharf/werrors"

	"github.com/itchio/httpkit/neterr"
	"github.com/itchio/httpkit/timeout"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/pkg/errors"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/cmd/wipe"
	"github.com/itchio/butler/database/models"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/hades"
)

var downloadsDriveCancelID = "Downloads.Drive"

const pingURL = "https://itch.io/static/ping.txt"

type Status struct {
	Online bool
}

func DownloadsDrive(rc *butlerd.RequestContext, params butlerd.DownloadsDriveParams) (*butlerd.DownloadsDriveResult, error) {
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
	messages.DownloadsDriveNetworkStatus.Notify(rc, butlerd.DownloadsDriveNetworkStatusNotification{
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
			} else {
				consumer.Warnf("Got non-network error while pinging: %+v", err)
			}
		} else {
			payload, _ := ioutil.ReadAll(res.Body)
			consumer.Statf("Looks like we're back online! (%s)", strings.TrimSpace(string(payload)))
			messages.DownloadsDriveNetworkStatus.Notify(rc, butlerd.DownloadsDriveNetworkStatusNotification{
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
	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustSelect(conn, &discardedDownloads, builder.Expr("discarded"), hades.Search{})
		models.PreloadDownloads(conn, discardedDownloads)
	})
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

		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustDelete(conn, download, builder.Eq{"id": download.ID})
		})

		messages.DownloadsDriveDiscarded.Notify(rc, butlerd.DownloadsDriveDiscardedNotification{
			Download: formatDownload(download),
		})
	}
	return nil
}

func performOne(parentCtx context.Context, rc *butlerd.RequestContext) error {
	consumer := rc.Consumer

	var pendingDownloads []*models.Download
	var download *models.Download
	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustSelect(conn, &pendingDownloads,
			builder.And(
				builder.IsNull{"finished_at"},
				builder.Not{builder.Expr("discarded")},
			),
			hades.Search{}.OrderBy("position ASC"),
		)
		if len(pendingDownloads) == 0 {
			return
		}

		download = pendingDownloads[0]
		download.Preload(conn)
	})
	if download == nil {
		return nil
	}
	consumer.Infof("%d pending downloads, performing for %s", len(pendingDownloads), operate.GameToString(download.Game))

	ctx, cancelFunc := context.WithCancel(parentCtx)
	defer cancelFunc()

	wasDiscarded := func() bool {
		// have we been discarded?
		{
			var discarded bool
			rc.WithConn(func(conn *sqlite.Conn) {
				models.MustExec(conn,
					builder.Select("discarded").From("downloads").Where(builder.Eq{"id": download.ID}),
					func(stmt *sqlite.Stmt) error {
						discarded = stmt.ColumnInt(0) == 1
						return nil
					},
				)
			})
			if discarded {
				consumer.Infof("Download was cancelled from under us, bailing out!")
				return true
			}
		}

		// has something else been prioritized?
		{
			var priorityDownloadID string
			rc.WithConn(func(conn *sqlite.Conn) {
				models.MustExecWithSearch(conn,
					builder.Select("id").From("downloads").Where(
						builder.And(
							builder.IsNull{"finished_at"},
							builder.Not{builder.Expr("discarded")},
						),
					),
					hades.Search{}.Limit(1),
					func(stmt *sqlite.Stmt) error {
						priorityDownloadID = stmt.ColumnText(0)
						return nil
					},
				)
			})
			if priorityDownloadID != download.ID {
				consumer.Infof("%s deprioritized (for %s), bailing out!", download.ID, priorityDownloadID)
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

		return messages.DownloadsDriveProgress.Notify(rc, butlerd.DownloadsDriveProgressNotification{
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
		params := paramsIn.(butlerd.ProgressNotification)
		progress = params.Progress
		eta = params.ETA
		bps = params.BPS
		return sendProgress()
	})

	defer rc.StopInterceptingNotification(messages.TaskStarted.Method())
	rc.InterceptNotification(messages.TaskStarted.Method(), func(method string, paramsIn interface{}) error {
		params := paramsIn.(butlerd.TaskStartedNotification)
		stage = string(params.Type)
		return sendProgress()
	})

	defer rc.StopInterceptingNotification(messages.TaskSucceeded.Method())
	rc.InterceptNotification(messages.TaskSucceeded.Method(), func(method string, paramsIn interface{}) error {
		return nil
	})

	err := func() (err error) {
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

		messages.DownloadsDriveStarted.Notify(rc, butlerd.DownloadsDriveStartedNotification{
			Download: formatDownload(download),
		})

		err = operate.InstallPerform(ctx, rc, butlerd.InstallPerformParams{
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
				rc.WithConn(func(conn *sqlite.Conn) {
					models.MustDelete(conn, &models.Download{}, builder.Eq{"id": download.ID})
				})
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
			} else if errors.Cause(err) == werrors.ErrCancelled {
				// just cancelled, nothing to see here
				return nil
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
		rc.WithConn(download.Save)

		messages.DownloadsDriveErrored.Notify(rc, butlerd.DownloadsDriveErroredNotification{
			Download: formatDownload(download),
		})

		return nil
	}

	consumer.Infof("Download finished!")
	finishedAt := time.Now().UTC()
	download.FinishedAt = &finishedAt
	rc.WithConn(download.Save)

	messages.DownloadsDriveFinished.Notify(rc, butlerd.DownloadsDriveFinishedNotification{
		Download: formatDownload(download),
	})

	return nil
}
