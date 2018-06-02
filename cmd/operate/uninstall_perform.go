package operate

import (
	"context"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/cmd/wipe"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
	"github.com/pkg/errors"
)

func UninstallPerform(ctx context.Context, rc *butlerd.RequestContext, params *butlerd.UninstallPerformParams) error {
	consumer := rc.Consumer

	cave := ValidateCave(rc, params.CaveID)
	var installFolder string
	rc.WithConn(func(conn *sqlite.Conn) {
		installFolder = cave.GetInstallFolder(conn)
	})

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
		// TODO: detect common uninstallers?
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

	err = messages.TaskStarted.Notify(rc, &butlerd.TaskStartedNotification{
		Reason: butlerd.TaskReasonUninstall,
		Type:   butlerd.TaskTypeUninstall,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	consumer.Infof("Running uninstall manager...")
	rc.StartProgress()
	err = manager.Uninstall(managerUninstallParams)
	rc.EndProgress()

	if err != nil {
		return errors.WithStack(err)
	}

	err = messages.TaskSucceeded.Notify(rc, &butlerd.TaskSucceededNotification{
		Type: butlerd.TaskTypeUninstall,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	consumer.Infof("Deleting cave...")
	rc.WithConn(cave.Delete)

	consumer.Infof("Wiping install folder...")
	err = wipe.Do(consumer, installFolder)
	if err != nil {
		return errors.WithStack(err)
	}

	consumer.Infof("Clearing out downloads...")
	rc.WithConn(func(conn *sqlite.Conn) {
		models.DiscardDownloadsByCaveID(conn, cave.ID)
	})

	return nil
}
