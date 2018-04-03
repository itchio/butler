package operate

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/itchio/go-itchio"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/wharf/eos"

	"github.com/itchio/butler/installer"

	"github.com/pkg/errors"
)

func InstallPerform(ctx context.Context, rc *butlerd.RequestContext, performParams *butlerd.InstallPerformParams) error {
	if performParams.StagingFolder == "" {
		return errors.New("No staging folder specified")
	}

	oc, err := LoadContext(ctx, rc, performParams.StagingFolder)
	if err != nil {
		return errors.WithStack(err)
	}

	meta := NewMetaSubcontext()
	oc.Load(meta)

	err = doInstallPerform(oc, meta)
	if err != nil {
		return errors.WithStack(err)
	}

	rc.Consumer.Infof("Install successful, retiring context")

	err = oc.Retire()
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func doInstallPerform(oc *OperationContext, meta *MetaSubcontext) error {
	rc := oc.rc
	params := meta.Data
	consumer := oc.Consumer()

	if !params.NoCave {
		cave := models.CaveByID(rc.DB(), params.CaveID)
		if cave == nil {
			cave = &models.Cave{
				ID:                params.CaveID,
				InstallFolderName: params.InstallFolderName,
				InstallLocationID: params.InstallLocationID,
			}
		}

		oc.cave = cave
	}

	client, err := ClientFromCredentials(params.Credentials)
	if err != nil {
		return errors.WithStack(err)
	}

	consumer.Infof("→ Installing %s", GameToString(params.Game))
	consumer.Infof("  (%s) is our destination", params.InstallFolder)
	consumer.Infof("  (%s) is our stage", oc.StageFolder())

	receiptIn, err := bfs.ReadReceipt(params.InstallFolder)
	if err != nil {
		receiptIn = nil
		consumer.Errorf("Could not read existing receipt: %s", err.Error())
	}

	if receiptIn == nil {
		consumer.Infof("No receipt found.")
	}

	istate := &InstallSubcontextState{}

	isub := &InstallSubcontext{
		Data: istate,
	}
	oc.Load(isub)

	// TODO: move that to queue
	if istate.DownloadSessionId == "" {
		res, err := client.NewDownloadSession(&itchio.NewDownloadSessionParams{
			GameID:        params.Game.ID,
			DownloadKeyID: params.Credentials.DownloadKey,
		})
		if err != nil {
			return errors.WithStack(err)
		}
		istate.DownloadSessionId = res.UUID
		oc.Save(isub)

		consumer.Infof("→ Starting fresh download session (%s)", istate.DownloadSessionId)
	} else {
		consumer.Infof("↻ Resuming download session (%s)", istate.DownloadSessionId)
	}

	if receiptIn == nil {
		consumer.Infof("← No previous install info (no recorded upload or build)")
	} else {
		consumer.Infof("← Previously installed:")
		LogUpload(consumer, receiptIn.Upload, receiptIn.Build)
	}

	consumer.Infof("→ To be installed:")
	LogUpload(consumer, params.Upload, params.Build)

	if receiptIn != nil && receiptIn.Upload != nil && receiptIn.Upload.ID == params.Upload.ID {
		consumer.Infof("Installing over same upload")
		if receiptIn.Build != nil && params.Build != nil {
			oldID := receiptIn.Build.ID
			newID := params.Build.ID
			if newID > oldID {
				consumer.Infof("↑ Upgrading from build %d to %d", oldID, newID)
				upgradePath, err := client.FindUpgrade(&itchio.FindUpgradeParams{
					CurrentBuildID: oldID,
					UploadID:       params.Upload.ID,
					DownloadKeyID:  params.Credentials.DownloadKey,
				})
				if err != nil {
					consumer.Warnf("Could not find upgrade path: %s", err.Error())
					consumer.Infof("Falling back to heal...")
					return heal(oc, meta, isub, receiptIn)
				}

				consumer.Infof("Found upgrade path with %d items: ", len(upgradePath.UpgradePath))
				var totalUpgradeSize int64
				for _, item := range upgradePath.UpgradePath {
					if item.ID == oldID {
						continue
					}

					consumer.Infof(" - Build %d (%s)", item.ID, humanize.IBytes(uint64(item.PatchSize)))
					totalUpgradeSize += item.PatchSize
				}
				fullUploadSize := params.Upload.Size

				var comparative = "smaller than"
				if totalUpgradeSize > fullUploadSize {
					comparative = "larger than"
				}
				consumer.Infof("Total upgrade size %s is %s full upload %s",
					humanize.IBytes(uint64(totalUpgradeSize)),
					comparative,
					humanize.IBytes(uint64(fullUploadSize)),
				)

				if totalUpgradeSize > fullUploadSize {
					consumer.Infof("Healing instead of patching")
					return heal(oc, meta, isub, receiptIn)
				}

				consumer.Warnf("TODO: update (falling back to install for now)")
			} else if newID < oldID {
				consumer.Infof("↓ Downgrading from build %d to %d", oldID, newID)
				return heal(oc, meta, isub, receiptIn)
			}

			consumer.Infof("↺ Re-installing build %d", newID)
			return heal(oc, meta, isub, receiptIn)
		}
	}

	installSourceURL := sourceURL(consumer, istate, params, "")

	// TODO: support http servers that don't have range request
	// (just copy it first). see DownloadInstallSource later on.
	file, err := eos.Open(installSourceURL)
	if err != nil {
		return errors.WithStack(err)
	}
	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		return errors.WithStack(err)
	}

	doForceLocal := func() (eos.File, error) {
		destName := filepath.Base(stats.Name())
		destPath := filepath.Join(oc.StageFolder(), "install-source", destName)

		if istate.IsAvailableLocally {
			consumer.Infof("Install source needs to be available locally, re-using previously-downloaded file")
		} else {
			consumer.Infof("Install source needs to be available locally, copying to disk...")

			dlErr := func() error {
				err = messages.TaskStarted.Notify(oc.rc, &butlerd.TaskStartedNotification{
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
				err := DownloadInstallSource(oc.Consumer(), oc.StageFolder(), oc.ctx, file, destPath)
				oc.rc.EndProgress()
				oc.consumer.Progress(0)
				if err != nil {
					return errors.WithStack(err)
				}

				err = messages.TaskSucceeded.Notify(oc.rc, &butlerd.TaskSucceededNotification{
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
			oc.Save(isub)
		}

		ret, err := eos.Open(destPath)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return ret, nil
	}

	if params.Build == nil && UploadIsProbablyExternal(params.Upload) {
		consumer.Warnf("Dealing with an external upload, all bets are off.")
		consumer.Warnf("Forcing download before we check anything else.")

		lf, err := doForceLocal()
		if err != nil {
			return errors.WithStack(err)
		}

		file.Close()
		file = lf
	}

	if istate.InstallerInfo == nil || istate.InstallerInfo.Type == installer.InstallerTypeUnknown {
		consumer.Infof("Determining source information...")

		installerInfo, err := installer.GetInstallerInfo(consumer, file)
		if err != nil {
			return errors.WithStack(err)
		}

		// sniffing may have read parts of the file, so seek back to beginning
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return errors.WithStack(err)
		}

		if params.IgnoreInstallers {
			switch installerInfo.Type {
			case installer.InstallerTypeArchive:
				// that's cool
			case installer.InstallerTypeNaked:
				// that's cool too
			default:
				consumer.Infof("Asked to ignore installers, forcing (naked) instead of (%s)", installerInfo.Type)
				installerInfo.Type = installer.InstallerTypeNaked
			}
		}

		istate.InstallerInfo = installerInfo
		oc.Save(isub)
	} else {
		consumer.Infof("Using cached source information")
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

		File:              file,
		InstallerInfo:     istate.InstallerInfo,
		StageFolderPath:   oc.StageFolder(),
		InstallFolderPath: params.InstallFolder,

		ReceiptIn: receiptIn,

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

		err = messages.TaskStarted.Notify(oc.rc, &butlerd.TaskStartedNotification{
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
			lf, localErr := doForceLocal()
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
		oc.Save(isub)
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
				oc.Save(isub)
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
}
