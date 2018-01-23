package operate

import (
	"fmt"
	"io"
	"net/url"
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

	consumer.Infof("→ Installing %s", gameToString(params.Game))
	consumer.Infof("  (%s) is our destination", params.InstallFolder)
	consumer.Infof("  (%s) is our stage", oc.StageFolder())

	client, err := clientFromCredentials(params.Credentials)
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
				logUpload(consumer, upload, upload.Build)
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
			logUpload(consumer, params.Upload, nil)
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

	if receiptIn == nil {
		consumer.Infof("← No previous install info (no recorded upload or build)")
	} else {
		consumer.Infof("← Previously installed:")
		logUpload(consumer, receiptIn.Upload, receiptIn.Build)
	}

	consumer.Infof("→ To be installed:")
	logUpload(consumer, params.Upload, params.Build)

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
					consumer.Warnf("TODO: heal")
				} else {
					consumer.Infof("Found upgrade path with %d items: ", len(upgradePath.UpgradePath))
					var totalUpgradeSize int64
					for _, item := range upgradePath.UpgradePath {
						consumer.Infof(" - Build %d (%s)", item.ID, humanize.IBytes(uint64(item.PatchSize)))
						totalUpgradeSize += item.PatchSize
					}

					var comparative = "smaller than"
					if totalUpgradeSize > params.Upload.Size {
						comparative = "larger than"
					}
					consumer.Infof("Total upgrade size %s is %s full upload %s",
						humanize.IBytes(uint64(totalUpgradeSize)),
						comparative,
						humanize.IBytes(uint64(params.Upload.Size)),
					)
					consumer.Warnf("TODO: update")
				}
			} else if newID < oldID {
				consumer.Infof("↓ Downgrading from build %d to %d", oldID, newID)
				consumer.Warnf("TODO: heal")
			} else {
				consumer.Infof("↺ Re-installing build %d", newID)
				consumer.Warnf("TODO: heal")
			}
		}
	}

	var installSourceURLPath string
	if params.Build == nil {
		installSourceURLPath = fmt.Sprintf("/upload/%d/download", params.Upload.ID)
	} else {
		fileType := "archive"

		for _, bf := range params.Build.Files {
			if bf.Type == itchio.BuildFileTypeUnpacked {
				consumer.Infof("Build %d / %d has an unpacked file", params.Upload.ID, params.Build.ID)
				fileType = "unpacked"
				break
			}
		}

		installSourceURLPath = fmt.Sprintf("/upload/%d/download/builds/%d/%s", params.Upload.ID, params.Build.ID, fileType)
	}
	values := make(url.Values)
	values.Set("api_key", params.Credentials.APIKey)
	if params.Credentials.DownloadKey != 0 {
		values.Set("download_key_id", fmt.Sprintf("%d", params.Credentials.DownloadKey))
	}
	var installSourceURL = fmt.Sprintf("itchfs://%s?%s", installSourceURLPath, values.Encode())

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

	istate := &InstallSubcontextState{}

	isub := &InstallSubcontext{
		data: istate,
	}

	oc.Load(isub)

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

		err = oc.conn.Notify(oc.ctx, "TaskSucceeded", &buse.TaskSucceededNotification{
			Type: buse.TaskTypeInstall,
			InstallResult: &buse.InstallResult{
				Game:   params.Game,
				Upload: params.Upload,
				Build:  params.Build,
			},
		})
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		return res, nil
	}

	res, err := tryInstall()
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

		res, err = tryInstall()
	}

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Infof("Install successful")
	if len(res.Files) == 1 {
		single := res.Files[0]

		consumer.Infof("Only installed a single file, is it an installer?")
		consumer.Infof("%s: probing", single)

		err = func() error {
			singlePath := filepath.Join(params.InstallFolder, single)
			sf, err := os.Open(singlePath)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			defer sf.Close()

			installerInfo, err = installer.GetInstallerInfo(consumer, sf)
			if err != nil {
				consumer.Infof("Could not determine installer info for single file, skipping: %s", err.Error())
				return nil
			}

			if !installer.IsWindowsInstaller(installerInfo.Type) {
				consumer.Infof("Installer type is (%s), ignoring", installerInfo.Type)
				return nil
			}

			consumer.Infof("Will use nested installer %s", installerInfo.Type)
			manager = installer.GetManager(string(installerInfo.Type))
			if manager == nil {
				return fmt.Errorf("Don't know how to install (%s) packages", installerInfo.Type)
			}

			sf.Close()

			destName := filepath.Base(single)
			destPath := filepath.Join(oc.StageFolder(), "nested-install-source", destName)

			_, err = os.Stat(destPath)
			if err == nil {
				// ah, it must already be there then
				consumer.Infof("%s: using for nested install", destPath)
			} else {
				consumer.Infof("%s: moving to", singlePath)
				consumer.Infof("%s - for nested install", destPath)

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
			res, err = tryInstall()
			return err
		}()
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	consumer.Infof("Writing receipt...")
	receipt := &bfs.Receipt{
		InstallerName: string(installerInfo.Type),
		Game:          params.Game,
		Upload:        params.Upload,
		Build:         params.Build,

		Files: res.Files,

		// optionals:
		MSIProductCode: res.MSIProductCode,
	}

	err = receipt.WriteReceipt(params.InstallFolder)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return res, nil
}

type InstallSubcontextState struct {
	InstallerInfo      *installer.InstallerInfo `json:"installerInfo,omitempty"`
	IsAvailableLocally bool                     `json:"isAvailableLocally"`
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
