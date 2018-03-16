package operate

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/butler/manager"
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

	consumer.Opf("Writing receipt...")
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
		// TODO: pass runtime in params?
		verdict, err := manager.Configure(consumer, params.InstallFolder, manager.CurrentRuntime())
		if err != nil {
			return errors.Wrap(err, 0)
		}

		consumer.Opf("Saving cave...")
		cave.SetVerdict(verdict)
		cave.InstalledSize = verdict.TotalSize
		cave.Game = params.Game
		cave.Upload = params.Upload
		cave.Build = params.Build
		cave.UpdateInstallTime()
		cave.Save(oc.rc.DB())
	}

	return nil
}
