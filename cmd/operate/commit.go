package operate

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
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

func commitInstall(oc *OperationContext, params *CommitInstallParams) (*installer.InstallResult, error) {
	consumer := oc.Consumer()

	res := params.InstallResult

	err := oc.conn.Notify(oc.ctx, "TaskSucceeded", &buse.TaskSucceededNotification{
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
		return nil, errors.Wrap(err, 0)
	}

	return res, nil
}
