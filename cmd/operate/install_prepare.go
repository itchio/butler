package operate

import (
	"io"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/headway/united"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"
	"github.com/pkg/errors"
)

type InstallPerformStrategy = int

const (
	InstallPerformStrategyNone    InstallPerformStrategy = 0
	InstallPerformStrategyInstall InstallPerformStrategy = 1
	InstallPerformStrategyHeal    InstallPerformStrategy = 2
	InstallPerformStrategyUpgrade InstallPerformStrategy = 3
)

type InstallPrepareResult struct {
	File      eos.File
	ReceiptIn *bfs.Receipt
	Strategy  InstallPerformStrategy
}

type InstallTask func(res *InstallPrepareResult) error

func InstallPrepare(oc *OperationContext, meta *MetaSubcontext, isub *InstallSubcontext, allowDownloads bool, task InstallTask) error {
	rc := oc.rc
	params := meta.Data
	consumer := oc.Consumer()

	client := rc.Client(params.Access.APIKey)

	res := &InstallPrepareResult{}

	receiptIn, err := bfs.ReadReceipt(params.InstallFolder)
	if err != nil {
		receiptIn = nil
		consumer.Errorf("Could not read existing receipt: %s", err.Error())
	}

	if receiptIn == nil {
		consumer.Infof("No receipt found.")
	}

	res.ReceiptIn = receiptIn

	istate := isub.Data

	if istate.DownloadSessionID == "" {
		res, err := client.NewDownloadSession(rc.Ctx, itchio.NewDownloadSessionParams{
			GameID:      params.Game.ID,
			Credentials: params.Access.Credentials,
		})
		if err != nil {
			return errors.WithStack(err)
		}
		istate.DownloadSessionID = res.UUID
		err = oc.Save(isub)
		if err != nil {
			return err
		}

		consumer.Infof("→ Starting fresh download session (%s)", istate.DownloadSessionID)
	} else {
		consumer.Infof("↻ Resuming download session (%s)", istate.DownloadSessionID)
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
				if istate.UsingHealFallback {
					consumer.Infof("Using heal fallback (as specified in install state)")
					res.Strategy = InstallPerformStrategyHeal
					return task(res)
				}

				upgradeRes, err := client.GetBuildUpgradePath(rc.Ctx, itchio.GetBuildUpgradePathParams{
					CurrentBuildID: oldID,
					TargetBuildID:  newID,
					Credentials:    params.Access.Credentials,
				})
				if err != nil {
					consumer.Warnf("Could not find upgrade path: %s", err.Error())
					consumer.Infof("Falling back to heal...")
					res.Strategy = InstallPerformStrategyHeal
					return task(res)
				}

				upgradePath := upgradeRes.UpgradePath
				// skip the current build, we're not interested in it
				upgradePath.Builds = upgradePath.Builds[1:]

				var totalUpgradeSize int64
				consumer.Infof("Found upgrade path with %d items: ", len(upgradePath.Builds))

				for _, b := range upgradePath.Builds {
					f := FindBuildFile(b.Files, itchio.BuildFileTypePatch, itchio.BuildFileSubTypeDefault)
					if f == nil {
						consumer.Warnf("Whoops, build %d is missing a patch, falling back to heal...", b.ID)
						res.Strategy = InstallPerformStrategyHeal
						return task(res)
					}

					{
						of := FindBuildFile(b.Files, itchio.BuildFileTypePatch, itchio.BuildFileSubTypeOptimized)
						if of != nil {
							f = of
						}
					}

					consumer.Infof(" - Build %d (%s)", b.ID, united.FormatBytes(f.Size))
					totalUpgradeSize += f.Size
				}
				fullUploadSize := params.Upload.Size

				var comparative = "smaller than"
				if totalUpgradeSize > fullUploadSize {
					comparative = "larger than"
				}
				consumer.Infof("Total upgrade size %s is %s full upload %s",
					united.FormatBytes(totalUpgradeSize),
					comparative,
					united.FormatBytes(fullUploadSize),
				)

				if totalUpgradeSize > fullUploadSize {
					consumer.Infof("Heal is less expensive, let's do that")
					res.Strategy = InstallPerformStrategyHeal
					return task(res)
				}

				consumer.Infof("Will apply %d patches", len(upgradePath.Builds))
				res.Strategy = InstallPerformStrategyUpgrade

				if istate.UpgradePath == nil {
					istate.UpgradePath = upgradePath
					istate.UpgradePathIndex = 0
					err = oc.Save(isub)
					if err != nil {
						return err
					}
				} else {
					consumer.Infof("%d patches already done, letting it resume", istate.UpgradePathIndex)
				}

				return task(res)
			} else if newID < oldID {
				consumer.Infof("↓ Downgrading from build %d to %d", oldID, newID)
				res.Strategy = InstallPerformStrategyHeal
				return task(res)
			}

			consumer.Infof("↺ Re-installing build %d", newID)
			res.Strategy = InstallPerformStrategyHeal
			return task(res)
		}
	}

	installSourceFileType := ""
	if params.IgnoreInstallers {
		installSourceFileType = "archive"
	}
	installSourceURL := MakeSourceURL(client, consumer, istate.DownloadSessionID, params, installSourceFileType)

	file, err := eos.Open(installSourceURL, option.WithConsumer(consumer))
	if err != nil {
		return errors.WithStack(err)
	}
	res.File = file
	defer file.Close()

	if params.Upload.Storage == itchio.UploadStorageExternal {
		consumer.Warnf("Dealing with an external upload (from %s), all bets are off.", params.Upload.Host)

		if IsBadExternalHost(params.Upload.Host) {
			consumer.Warnf("Host (%s) is known not to work, failing early.", params.Upload.Host)
			return errors.WithStack(butlerd.CodeUnsupportedHost)
		}

		if !allowDownloads {
			consumer.Warnf("Can't determine source information at that time")
			return nil
		}

		consumer.Warnf("Forcing download before we check anything else.")
		lf, err := doForceLocal(file, oc, meta, isub)
		if err != nil {
			return errors.WithStack(err)
		}

		file.Close()
		file = lf
		res.File = lf
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

		dui, err := AssessDiskUsage(file, receiptIn, params.InstallFolder, installerInfo)
		if err != nil {
			return errors.WithMessage(err, "assessing disk usage")
		}

		consumer.Infof("Estimated disk usage (accuracy: %s)", dui.Accuracy)
		consumer.Infof("  ✓ %s needed free space", united.FormatBytes(dui.NeededFreeSpace))
		consumer.Infof("  ✓ %s final disk usage", united.FormatBytes(dui.FinalDiskUsage))

		istate.InstallerInfo = installerInfo
		err = oc.Save(isub)
		if err != nil {
			return err
		}
	} else {
		consumer.Infof("Using cached source information")
	}

	installerInfo := istate.InstallerInfo
	if installerInfo.Type == installer.InstallerTypeUnsupported {
		consumer.Errorf("Item is packaged in a way that isn't supported, refusing to install")
		return errors.WithStack(butlerd.CodeUnsupportedPackaging)
	}

	return task(res)
}
