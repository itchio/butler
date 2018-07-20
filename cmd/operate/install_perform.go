package operate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/eos/option"

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
			err = DownloadInstallSource(oc.Consumer(), oc.StageFolder(), oc.ctx, file, destPath)
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

	consumer.Infof("→ Performing install for %s", GameToString(params.Game))
	consumer.Infof("    to (%s)", params.InstallFolder)
	consumer.Infof("    via (%s)", oc.StageFolder())

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

		if prepareRes.Strategy == InstallPerformStrategyHeal {
			return heal(oc, meta, isub, prepareRes.ReceiptIn)
		}

		if prepareRes.Strategy == InstallPerformStrategyUpgrade {
			return upgrade(oc, meta, isub, prepareRes.ReceiptIn)
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

		managerInstallParams := &installer.InstallParams{
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

		var firstInstallResult = istate.FirstInstallResult

		if firstInstallResult != nil {
			consumer.Infof("First install already completed (%d files)", len(firstInstallResult.Files))
		} else {
			var err error
			firstInstallResult, err = tryInstall()
			if err != nil && errors.Cause(err) == installer.ErrNeedLocal {
				lf, localErr := doForceLocal(prepareRes.File, oc, meta, isub)
				if localErr != nil {
					return errors.WithStack(err)
				}

				consumer.Infof("Re-invoking manager with local file...")
				managerInstallParams.File = lf

				firstInstallResult, err = tryInstall()
			}
			if err != nil {
				return errors.WithStack(err)
			}

			consumer.Infof("Install successful")

			istate.FirstInstallResult = firstInstallResult
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

		var finalInstallResult = firstInstallResult
		var finalInstallerInfo = installerInfo

		if len(firstInstallResult.Files) == 1 {
			single := firstInstallResult.Files[0]
			singlePath := filepath.Join(params.InstallFolder, single)

			consumer.Infof("Installed a single file")

			err = func() error {
				secondInstallerInfo := istate.SecondInstallerInfo
				if secondInstallerInfo != nil {
					consumer.Infof("Using cached second installer info")
				} else {
					consumer.Infof("Probing (%s)...", single)
					sf, err := os.Open(singlePath)
					if err != nil {
						return errors.WithStack(err)
					}
					defer sf.Close()

					secondInstallerInfo, err = installer.GetInstallerInfo(consumer, sf)
					if err != nil {
						consumer.Infof("Could not determine installer info for single file, skipping: %s", err.Error())
						return nil
					}

					sf.Close()

					istate.SecondInstallerInfo = secondInstallerInfo
					err = oc.Save(isub)
					if err != nil {
						return err
					}
				}

				if !installer.IsWindowsInstaller(secondInstallerInfo.Type) {
					consumer.Infof("Installer type is (%s), ignoring", secondInstallerInfo.Type)
					return nil
				}

				consumer.Infof("Will use nested installer (%s)", secondInstallerInfo.Type)
				finalInstallerInfo = secondInstallerInfo
				manager = installer.GetManager(string(secondInstallerInfo.Type))
				if manager == nil {
					return fmt.Errorf("Don't know how to install (%s) packages", secondInstallerInfo.Type)
				}

				destName := filepath.Base(single)
				destPath := filepath.Join(oc.StageFolder(), "nested-install-source", destName)

				_, err = os.Stat(destPath)
				if err == nil {
					// ah, it must already be there then
					consumer.Infof("Using (%s) for nested install", destPath)
				} else {
					consumer.Infof("Moving (%s) to (%s) for nested install", singlePath, destPath)

					err = os.MkdirAll(filepath.Dir(destPath), 0755)
					if err != nil {
						return errors.WithStack(err)
					}

					err = os.RemoveAll(destPath)
					if err != nil {
						return errors.WithStack(err)
					}

					err = os.Rename(singlePath, destPath)
					if err != nil {
						return errors.WithStack(err)
					}
				}

				lf, err := os.Open(destPath)
				if err != nil {
					return errors.WithStack(err)
				}

				managerInstallParams.File = lf

				consumer.Infof("Invoking nested install manager, let's go!")
				finalInstallResult, err = tryInstall()
				return err
			}()
			if err != nil {
				return errors.WithStack(err)
			}
		}

		return commitInstall(oc, &CommitInstallParams{
			InstallFolder: params.InstallFolder,

			InstallerName: string(finalInstallerInfo.Type),
			Game:          params.Game,
			Upload:        params.Upload,
			Build:         params.Build,

			InstallResult: finalInstallResult,
		})

	})
}
