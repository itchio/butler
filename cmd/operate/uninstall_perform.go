package operate

import (
	"context"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/cmd/wipe"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
)

func UninstallPerform(ctx context.Context, rc *buse.RequestContext, params *buse.UninstallPerformParams) error {
	consumer := rc.Consumer

	cave := ValidateCave(rc, params.CaveID)
	installFolder := cave.GetInstallFolder(rc.DB())

	consumer.Infof("→ Uninstalling %s", installFolder)

	var installerType = installer.InstallerTypeUnknown

	receipt, err := bfs.ReadReceipt(installFolder)
	if err != nil {
		consumer.Warnf("Could not read receipt: %s", err.Error())
	}

	if receipt != nil && receipt.InstallerName != "" {
		installerType = (installer.InstallerType)(receipt.InstallerName)
	}

	consumer.Infof("Will use installer %s", installerType)
	manager := installer.GetManager(string(installerType))
	if manager == nil {
		consumer.Warnf("No manager for installer %s", installerType)
		consumer.Infof("Falling back to archive")

		manager = installer.GetManager("archive")
		if manager == nil {
			return errors.New("archive install manager not found, can't uninstall")
		}
	}

	managerUninstallParams := &installer.UninstallParams{
		InstallFolderPath: installFolder,
		Consumer:          consumer,
		Receipt:           receipt,
	}

	err = messages.TaskStarted.Notify(rc, &buse.TaskStartedNotification{
		Reason: buse.TaskReasonUninstall,
		Type:   buse.TaskTypeUninstall,
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Infof("Running uninstall manager...")
	rc.StartProgress()
	err = manager.Uninstall(managerUninstallParams)
	rc.EndProgress()

	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = messages.TaskSucceeded.Notify(rc, &buse.TaskSucceededNotification{
		Type: buse.TaskTypeUninstall,
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Infof("Deleting cave...")
	err = rc.DB().Delete(cave).Error
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Infof("Wiping install folder...")
	err = wipe.Do(consumer, installFolder)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Infof("Clearing out downloads...")
	err = rc.DB().Model(&models.Download{}).Where("cave_id = ?", cave.ID).Update("discarded", true).Error
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
