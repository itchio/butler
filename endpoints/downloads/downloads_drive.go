package downloads

import (
	"context"
	"time"

	"github.com/go-errors/errors"

	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
)

var downloadsDriveCancelID = "Downloads.Drive"

func DownloadsDrive(rc *buse.RequestContext, params *buse.DownloadsDriveParams) (*buse.DownloadsDriveResult, error) {
	consumer := rc.Consumer
	consumer.Infof("Now driving downloads...")

	parentCtx := rc.Ctx
	ctx, cancelFunc := context.WithCancel(parentCtx)

	rc.CancelFuncs.Add(downloadsDriveCancelID, cancelFunc)
	defer rc.CancelFuncs.Remove(downloadsDriveCancelID)

poll:
	for {
		select {
		case <-ctx.Done():
			consumer.Infof("Drive cancelled, bye!")
			break poll
		default:
			// let's keep going
		}

		err := performOne(ctx, rc)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		time.Sleep(1 * time.Second)
	}

	res := &buse.DownloadsDriveResult{}
	return res, nil
}

func performOne(ctx context.Context, rc *buse.RequestContext) error {
	consumer := rc.Consumer

	var pendingDownloads []*models.Download
	err := rc.DB().Where(`finished_at IS NULL AND error IS NULL`).Order(`position ASC`).Find(&pendingDownloads).Error
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if len(pendingDownloads) == 0 {
		return nil
	}

	download := pendingDownloads[0]
	download.Preload(rc.DB())
	consumer.Infof("%d pending downloads, performing for %s", len(pendingDownloads), operate.GameToString(download.Game))

	var stage = "prepare"
	var progress, eta, bps float64

	sendProgress := func() error {
		return messages.DownloadsDriveProgress.Notify(rc, &buse.DownloadsDriveProgressNotification{
			Download: formatDownload(download),
			Progress: &buse.DownloadProgress{
				Stage:    stage,
				Progress: progress,
				ETA:      eta,
				BPS:      bps,
			},
		})
	}

	defer rc.StopInterceptingNotification(messages.Progress.Method())
	rc.InterceptNotification(messages.Progress.Method(), func(method string, paramsIn interface{}) error {
		params := paramsIn.(*buse.ProgressNotification)
		progress = params.Progress
		eta = params.ETA
		bps = params.BPS
		return sendProgress()
	})

	defer rc.StopInterceptingNotification(messages.TaskStarted.Method())
	rc.InterceptNotification(messages.TaskStarted.Method(), func(method string, paramsIn interface{}) error {
		params := paramsIn.(*buse.TaskStartedNotification)
		stage = string(params.Type)
		return sendProgress()
	})

	err = func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				consumer.Warnf("Recovered from panic!")
				if rErr, ok := r.(error); ok {
					err = errors.Wrap(rErr, 0)
				} else {
					err = errors.New(r)
				}
			}
		}()

		err = operate.InstallPerform(ctx, rc, &buse.InstallPerformParams{
			ID:            download.ID,
			StagingFolder: download.StagingFolder,
		})
		return
	}()
	if err != nil {
		if be, ok := buse.AsBuseError(err); ok {
			switch buse.Code(be.AsJsonRpc2().Code) {
			case buse.CodeOperationCancelled:
				// the whole drive was probably cancelled?
				return nil
			case buse.CodeOperationAborted:
				consumer.Warnf("Download aborted, cleaning it out.")
				err := rc.DB().Delete(download).Error
				if err != nil {
					return errors.Wrap(err, 0)
				}
				return nil
			}
		}

		var errString = err.Error()
		if se, ok := err.(*errors.Error); ok {
			errString = se.ErrorStack()
		}

		consumer.Warnf("Download failed: %s", errString)
		download.Error = &errString
		download.Save(rc.DB())
		return nil
	}

	consumer.Infof("Download finished!")
	finishedAt := time.Now().UTC()
	download.FinishedAt = &finishedAt
	download.Save(rc.DB())

	return nil
}
