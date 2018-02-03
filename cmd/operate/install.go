package operate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/itchio/go-itchio"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/wharf/eos"

	"github.com/itchio/butler/installer"

	"github.com/go-errors/errors"
)

func install(oc *OperationContext, meta *MetaSubcontext) (*installer.InstallResult, error) {
	consumer := oc.Consumer()

	params := meta.data.InstallParams

	if params == nil {
		return nil, errors.New("Missing install params")
	}

	if params.Game == nil {
		return nil, errors.New("Missing game in install")
	}

	if params.InstallFolder == "" {
		return nil, errors.New("Missing install folder in install")
	}

	consumer.Infof("→ Installing %s", GameToString(params.Game))
	consumer.Infof("  (%s) is our destination", params.InstallFolder)
	consumer.Infof("  (%s) is our stage", oc.StageFolder())

	client, err := ClientFromCredentials(params.Credentials)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if params.Upload == nil {
		consumer.Infof("No upload specified, looking for compatible ones...")
		uploadsFilterResult, err := getFilteredUploads(client, params.Game, params.Credentials, consumer)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		if len(uploadsFilterResult.Uploads) == 0 {
			consumer.Errorf("Didn't find a compatible upload.")
			consumer.Errorf("The initial %d uploads were:", len(uploadsFilterResult.InitialUploads))
			for _, upload := range uploadsFilterResult.InitialUploads {
				LogUpload(consumer, upload, upload.Build)
			}

			return nil, (&OperationError{
				Code:      "noCompatibleUploads",
				Message:   "No compatible uploads",
				Operation: "install",
			}).Throw()
		}

		if len(uploadsFilterResult.Uploads) == 1 {
			params.Upload = uploadsFilterResult.Uploads[0]
		} else {
			var r buse.PickUploadResult
			err := oc.conn.Call(oc.ctx, "PickUpload", &buse.PickUploadParams{
				Uploads: uploadsFilterResult.Uploads,
			}, &r)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}

			if r.Index < 0 {
				return nil, ErrAborted
			}

			params.Upload = uploadsFilterResult.Uploads[r.Index]
		}

		if params.Upload.Build != nil {
			// if we reach this point, we *just now* queried for an upload,
			// so we know the build object is the latest
			params.Build = params.Upload.Build
		}

		oc.Save(meta)
	}

	// params.Upload can't be nil by now
	if params.Build == nil {
		// We were passed an upload but not a build:
		// Let's refresh upload info so we can settle on a build we want to install (if any)

		listUploadsRes, err := client.ListGameUploads(&itchio.ListGameUploadsParams{
			GameID:        params.Game.ID,
			DownloadKeyID: params.Credentials.DownloadKey,
		})
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		found := true
		for _, u := range listUploadsRes.Uploads {
			if u.ID == params.Upload.ID {
				if u.Build == nil {
					consumer.Infof("Upload is not wharf-enabled")
				} else {
					consumer.Infof("Latest build for upload is %d", u.Build.ID)
					params.Build = u.Build
				}
				break
			}
		}

		if !found {
			consumer.Errorf("Uh oh, we didn't find that upload on the server:")
			LogUpload(consumer, params.Upload, nil)
			return nil, errors.New("Upload not found")
		}

		oc.Save(meta)
	}

	receiptIn, err := bfs.ReadReceipt(params.InstallFolder)
	if err != nil {
		receiptIn = nil
		consumer.Errorf("Could not read existing receipt: %s", err.Error())
	}

	if receiptIn == nil {
		consumer.Infof("No receipt found, asking client for info...")

		var r buse.GetReceiptResult
		err := oc.conn.Call(oc.ctx, "GetReceipt", &buse.GetReceiptParams{}, &r)
		if err != nil {
			consumer.Warnf("Could not get receipt from client: %s", err.Error())
		}

		if r.Receipt == nil {
			consumer.Infof("Client returned nil receipt")
		} else {
			consumer.Infof("Got receipt from client")
			receiptIn = r.Receipt
		}
	}

	istate := &InstallSubcontextState{}

	isub := &InstallSubcontext{
		data: istate,
	}
	oc.Load(isub)

	if istate.DownloadSessionId == "" {
		res, err := client.NewDownloadSession(&itchio.NewDownloadSessionParams{
			GameID:        params.Game.ID,
			DownloadKeyID: params.Credentials.DownloadKey,
		})
		if err != nil {
			return nil, errors.Wrap(err, 0)
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
		return nil, errors.Wrap(err, 0)
	}
	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if istate.InstallerInfo == nil {
		consumer.Infof("Determining source information...")

		installerInfo, err := installer.GetInstallerInfo(consumer, file)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		// sniffing may have read parts of the file, so seek back to beginning
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return nil, errors.Wrap(err, 0)
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
		return nil, errors.New(msg)
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
			return nil, ErrCancelled
		default:
			// keep going!
		}

		err = oc.conn.Notify(oc.ctx, "TaskStarted", &buse.TaskStartedNotification{
			Reason:    buse.TaskReasonInstall,
			Type:      buse.TaskTypeInstall,
			Game:      params.Game,
			Upload:    params.Upload,
			Build:     params.Build,
			TotalSize: stats.Size(),
		})
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		oc.StartProgress()
		res, err := manager.Install(managerInstallParams)
		oc.EndProgress()

		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		return res, nil
	}

	var firstInstallResult *installer.InstallResult
	firstInstallResult = istate.FirstInstallResult

	if firstInstallResult != nil {
		consumer.Infof("First install already completed (%d files)", len(firstInstallResult.Files))
	} else {
		var err error
		firstInstallResult, err = tryInstall()
		if err != nil && errors.Is(err, installer.ErrNeedLocal) {
			destName := filepath.Base(stats.Name())
			destPath := filepath.Join(oc.StageFolder(), "install-source", destName)

			if istate.IsAvailableLocally {
				consumer.Infof("Install source needs to be available locally, re-using previously-downloaded file")
			} else {
				consumer.Infof("Install source needs to be available locally, copying to disk...")

				dlErr := func() error {
					err = oc.conn.Notify(oc.ctx, "TaskStarted", &buse.TaskStartedNotification{
						Reason:    buse.TaskReasonInstall,
						Type:      buse.TaskTypeDownload,
						Game:      params.Game,
						Upload:    params.Upload,
						Build:     params.Build,
						TotalSize: stats.Size(),
					})
					if err != nil {
						return errors.Wrap(err, 0)
					}

					oc.StartProgress()
					err := DownloadInstallSource(oc, file, destPath)
					oc.EndProgress()
					oc.consumer.Progress(0)
					if err != nil {
						return errors.Wrap(err, 0)
					}

					err = oc.conn.Notify(oc.ctx, "TaskSucceeded", &buse.TaskSucceededNotification{
						Type: buse.TaskTypeDownload,
					})
					if err != nil {
						return errors.Wrap(err, 0)
					}
					return nil
				}()

				if dlErr != nil {
					return nil, errors.Wrap(dlErr, 0)
				}

				istate.IsAvailableLocally = true
				oc.Save(isub)
			}

			consumer.Infof("Re-invoking manager with local file...")
			{
				lf, err := os.Open(destPath)
				if err != nil {
					return nil, errors.Wrap(err, 0)
				}
				managerInstallParams.File = lf
			}

			firstInstallResult, err = tryInstall()
		}

		if err != nil {
			return nil, errors.Wrap(err, 0)
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
					return errors.Wrap(err, 0)
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
					return errors.Wrap(err, 0)
				}

				err = os.RemoveAll(destPath)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				err = os.Rename(singlePath, destPath)
				if err != nil {
					return errors.Wrap(err, 0)
				}
			}

			lf, err := os.Open(destPath)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			managerInstallParams.File = lf

			consumer.Infof("Invoking nested install manager, let's go!")
			finalInstallResult, err = tryInstall()
			return err
		}()
		if err != nil {
			return nil, errors.Wrap(err, 0)
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

type InstallSubcontextState struct {
	DownloadSessionId   string                   `json:"downloadSessionId,omitempty"`
	InstallerInfo       *installer.InstallerInfo `json:"installerInfo,omitempty"`
	IsAvailableLocally  bool                     `json:"isAvailableLocally,omitempty"`
	FirstInstallResult  *installer.InstallResult `json:"firstInstallResult,omitempty"`
	SecondInstallerInfo *installer.InstallerInfo `json:"secondInstallerInfo,omitempty"`
}

type InstallSubcontext struct {
	data *InstallSubcontextState
}

var _ Subcontext = (*InstallSubcontext)(nil)

func (mt *InstallSubcontext) Key() string {
	return "install"
}

func (mt *InstallSubcontext) Data() interface{} {
	return &mt.data
}
