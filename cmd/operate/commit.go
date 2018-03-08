package operate

import (
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/configurator"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
	itchio "github.com/itchio/go-itchio"
)

type CommitInstallParams struct {
	InstallerName string
	InstallFolder string

	Game   *itchio.Game
	Upload *itchio.Upload
	Build  *itchio.Build

	InstallResult *installer.InstallResult
}

func commitInstall(oc *OperationContext, params *CommitInstallParams) error {
	consumer := oc.Consumer()

	res := params.InstallResult

	err := messages.TaskSucceeded.Notify(oc.rc, &buse.TaskSucceededNotification{
		Type: buse.TaskTypeInstall,
		InstallResult: &buse.InstallResult{
			Game:   params.Game,
			Upload: params.Upload,
			Build:  params.Build,
		},
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Infof("Writing receipt...")
	receipt := &bfs.Receipt{
		InstallerName: params.InstallerName,
		Game:          params.Game,
		Upload:        params.Upload,
		Build:         params.Build,

		Files: res.Files,

		// optionals:
		MSIProductCode: res.MSIProductCode,
	}

	err = receipt.WriteReceipt(params.InstallFolder)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	cave := oc.cave
	if cave != nil {
		consumer.Infof("Configuring...")
		verdict, err := configurator.Configure(params.InstallFolder, false)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		consumer.Infof("Saving cave...")
		cave.SetVerdict(verdict)
		cave.InstalledSize = verdict.TotalSize
		cave.Game = params.Game
		cave.Upload = params.Upload
		cave.Build = params.Build
		cave.InstalledAt = time.Now().UTC()

		cave.Save(oc.rc.DB())
	}

	return nil
}
