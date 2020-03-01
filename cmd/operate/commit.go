package operate

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/manager"
	"github.com/itchio/hush"
	"github.com/itchio/hush/bfs"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/ox"
	"github.com/pkg/errors"
)

type CommitInstallParams struct {
	InstallerName string
	InstallFolder string

	Game   *itchio.Game
	Upload *itchio.Upload
	Build  *itchio.Build

	InstallResult *hush.InstallResult
}

func commitInstall(oc *OperationContext, params *CommitInstallParams) error {
	consumer := oc.Consumer()

	res := params.InstallResult

	err := messages.TaskSucceeded.Notify(oc.rc, butlerd.TaskSucceededNotification{
		Type: butlerd.TaskTypeInstall,
		InstallResult: &butlerd.InstallResult{
			Game:   params.Game,
			Upload: params.Upload,
			Build:  params.Build,
		},
	})
	if err != nil {
		return errors.WithStack(err)
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
		return errors.WithStack(err)
	}

	cave := oc.cave
	if cave != nil {
		// TODO: pass runtime in params?
		verdict, err := manager.Configure(consumer, params.InstallFolder, ox.CurrentRuntime())
		if err != nil {
			return errors.WithStack(err)
		}

		consumer.Opf("Saving cave...")
		cave.SetVerdict(verdict)
		cave.InstalledSize = verdict.TotalSize
		cave.Game = params.Game
		cave.Upload = params.Upload
		cave.Build = params.Build
		cave.UpdateInstallTime()
		oc.rc.WithConn(cave.SaveWithAssocs)
	}

	return nil
}
