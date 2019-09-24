package operate

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/itchio/butler/manager/runlock"
	itchio "github.com/itchio/go-itchio"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"
	"github.com/itchio/wharf/pwr/patcher"

	"github.com/itchio/butler/installer"

	"github.com/pkg/errors"
)

func InstallPerform(ctx context.Context, rc *butlerd.RequestContext, performParams butlerd.InstallPerformParams) error {
	if performParams.StagingFolder == "" {
		return errors.New("No staging folder specified")
	}

	oc, err := LoadContext(ctx, rc, performParams.StagingFolder)
	if err != nil {
		return errors.WithStack(err)
	}
	defer oc.Release()

	meta := NewMetaSubcontext()
	oc.Load(meta)

	err = doInstallPerform(oc, meta)
	if err != nil {
		oc.Consumer().Errorf("%+v", err)
		return errors.WithStack(err)
	}

	oc.Retire()

	return nil
}

func doForceLocal(file eos.File, oc *OperationContext, meta *MetaSubcontext, isub *InstallSubcontext) (eos.File, error) {
	consumer := oc.rc.Consumer
	params := meta.Data
	istate := isub.Data

	stats, err := file.Stat()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	destName := filepath.Base(stats.Name())
	destPath := filepath.Join(oc.StageFolder(), "install-source", destName)

	if istate.IsAvailableLocally {
		consumer.Infof("Install source needs to be available locally, re-using previously-downloaded file")
	} else {
		consumer.Infof("Install source needs to be available locally, copying to disk...")

		dlErr := func() error {
			err := messages.TaskStarted.Notify(oc.rc, butlerd.TaskStartedNotification{
				Reason:    butlerd.TaskReasonInstall,
				Type:      butlerd.TaskTypeDownload,
				Game:      params.Game,
				Upload:    params.Upload,
				Build:     params.Build,
				TotalSize: stats.Size(),
			})
			if err != nil {
				return errors.WithStack(err)
			}

			oc.rc.StartProgress()
			err = DownloadInstallSource(DownloadInstallSourceParams{
				Context:       oc.ctx,
				Consumer:      oc.Consumer(),
				StageFolder:   oc.StageFolder(),
				OperationName: "force-local",
				File:          file,
				DestPath:      destPath,
			})
			oc.rc.EndProgress()
			oc.consumer.Progress(0)
			if err != nil {
				return errors.WithStack(err)
			}

			err = messages.TaskSucceeded.Notify(oc.rc, butlerd.TaskSucceededNotification{
				Type: butlerd.TaskTypeDownload,
			})
			if err != nil {
				return errors.WithStack(err)
			}
			return nil
		}()

		if dlErr != nil {
			return nil, errors.Wrap(dlErr, "downloading install source")
		}

		istate.IsAvailableLocally = true
		err = oc.Save(isub)
		if err != nil {
			return nil, err
		}
	}

	ret, err := eos.Open(destPath, option.WithConsumer(consumer))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return ret, nil
}

func doInstallPerform(oc *OperationContext, meta *MetaSubcontext) error {
	rc := oc.rc
	params := meta.Data
	consumer := oc.Consumer()

	istate := &InstallSubcontextState{}
	isub := &InstallSubcontext{
		Data: istate,
	}
	oc.Load(isub)

	if params.Game == nil {
		return errors.Errorf("Corrupted download info (missing game), refusing to continue.")
	}

	{
		if !istate.RefreshedGame {
			client := rc.Client(params.Access.APIKey)
			istate.RefreshedGame = true
			oc.Save(isub)

			// attempt to refresh game info
			gameRes, err := client.GetGame(rc.Ctx, itchio.GetGameParams{
				GameID:      params.Game.ID,
				Credentials: params.Access.Credentials,
			})
			if err != nil {
				consumer.Warnf("Could not refresh game info: %s", err.Error())
			} else {
				params.Game = gameRes.Game
				oc.Save(meta)
			}
		}
	}

	consumer.Infof("→ Performing install for %s", GameToString(params.Game))
	consumer.Infof("    to (%s)", params.InstallFolder)
	consumer.Infof("    via (%s)", oc.StageFolder())

	rlock := runlock.New(consumer, params.InstallFolder)
	err := rlock.Lock(rc.Ctx, "install")
	if err != nil {
		return errors.WithStack(err)
	}
	defer rlock.Unlock()

	return InstallPrepare(oc, meta, isub, true, func(prepareRes *InstallPrepareResult) error {
		if !params.NoCave {
			var cave *models.Cave
			rc.WithConn(func(conn *sqlite.Conn) {
				cave = models.CaveByID(conn, params.CaveID)
			})
			if cave == nil {
				cave = &models.Cave{
					ID:                params.CaveID,
					InstallFolderName: params.InstallFolderName,
					InstallLocationID: params.InstallLocationID,
				}
			}

			oc.cave = cave
		}

		if prepareRes.Strategy == InstallPerformStrategyUpgrade {
			err := upgrade(oc, meta, isub, prepareRes.ReceiptIn)
			if err == nil || errors.Cause(err) == patcher.ErrStop {
				return err
			}

			consumer := oc.Consumer()
			consumer.Warnf("Patching failed: %+v", err)

			consumer.Warnf("Falling back to heal...")
			istate.UsingHealFallback = true
			oc.Save(isub)
			prepareRes.Strategy = InstallPerformStrategyHeal
		}

		if prepareRes.Strategy == InstallPerformStrategyHeal {
			return heal(oc, meta, isub, prepareRes.ReceiptIn)
		}

		stats, err := prepareRes.File.Stat()
		if err != nil {
			return errors.WithStack(err)
		}

		installerInfo := istate.InstallerInfo

		consumer.Infof("Will use installer %s", installerInfo.Type)
		manager := installer.GetManager(string(installerInfo.Type))
		if manager == nil {
			msg := fmt.Sprintf("No manager for installer %s", installerInfo.Type)
			return errors.New(msg)
		}

		managerInstallParams := installer.InstallParams{
			Consumer: consumer,

			File:              prepareRes.File,
			InstallerInfo:     istate.InstallerInfo,
			StageFolderPath:   oc.StageFolder(),
			InstallFolderPath: params.InstallFolder,

			ReceiptIn: prepareRes.ReceiptIn,

			Context: oc.ctx,
		}

		tryInstall := func() (*installer.InstallResult, error) {
			defer managerInstallParams.File.Close()

			select {
			case <-oc.ctx.Done():
				return nil, errors.WithStack(butlerd.CodeOperationCancelled)
			default:
				// keep going!
			}

			err = messages.TaskStarted.Notify(oc.rc, butlerd.TaskStartedNotification{
				Reason:    butlerd.TaskReasonInstall,
				Type:      butlerd.TaskTypeInstall,
				Game:      params.Game,
				Upload:    params.Upload,
				Build:     params.Build,
				TotalSize: stats.Size(),
			})
			if err != nil {
				return nil, errors.WithStack(err)
			}

			oc.rc.StartProgress()
			res, err := manager.Install(managerInstallParams)
			oc.rc.EndProgress()

			if err != nil {
				return nil, errors.WithStack(err)
			}

			return res, nil
		}

		var installResult = istate.FirstInstallResult

		if installResult != nil {
			consumer.Infof("First install already completed (%d files)", len(installResult.Files))
		} else {
			var err error
			installResult, err = tryInstall()
			if err != nil && errors.Cause(err) == installer.ErrNeedLocal {
				lf, localErr := doForceLocal(prepareRes.File, oc, meta, isub)
				if localErr != nil {
					return errors.WithStack(err)
				}

				consumer.Infof("Re-invoking manager with local file...")
				managerInstallParams.File = lf

				installResult, err = tryInstall()
			}
			if err != nil {
				return errors.WithStack(err)
			}

			consumer.Infof("Install successful")

			istate.FirstInstallResult = installResult
			err = oc.Save(isub)
			if err != nil {
				return err
			}
		}

		select {
		case <-oc.ctx.Done():
			consumer.Warnf("Asked to cancel, so, cancelling...")
			return errors.WithStack(butlerd.CodeOperationCancelled)
		default:
			// continue!
		}

		return commitInstall(oc, &CommitInstallParams{
			InstallFolder: params.InstallFolder,

			InstallerName: string(installerInfo.Type),
			Game:          params.Game,
			Upload:        params.Upload,
			Build:         params.Build,

			InstallResult: installResult,
		})

	})
}
