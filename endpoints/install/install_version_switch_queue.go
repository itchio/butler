package install

import (
	"fmt"

	"github.com/itchio/butler/butlerd/messages"
	itchio "github.com/itchio/go-itchio"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/pkg/errors"
)

func InstallVersionSwitchQueue(rc *butlerd.RequestContext, params *butlerd.InstallVersionSwitchQueueParams) (*butlerd.InstallVersionSwitchQueueResult, error) {
	consumer := rc.Consumer

	cave := operate.ValidateCave(rc, params.CaveID)

	consumer.Infof("Looking for other versions of %s", operate.GameToString(cave.Game))

	upload := cave.Upload
	if upload == nil {
		return nil, fmt.Errorf("No other versions available for %s", operate.GameToString(cave.Game))
	}

	credentials := operate.CredentialsForGameID(rc.DB(), cave.Game.ID)

	client, err := operate.ClientFromCredentials(credentials)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	buildsRes, err := client.ListUploadBuilds(&itchio.ListUploadBuildsParams{
		UploadID:      upload.ID,
		DownloadKeyID: credentials.DownloadKey,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	pickRes, err := messages.InstallVersionSwitchPick.Call(rc, &butlerd.InstallVersionSwitchPickParams{
		Upload: upload,
		Builds: buildsRes.Builds,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if pickRes.Index < 0 {
		return nil, errors.WithStack(butlerd.CodeOperationAborted)
	}

	build := buildsRes.Builds[pickRes.Index]

	if true {
		return nil, errors.New("We're up to InstallQueue and butler doesn't fully handle downloads yet :o")
	}

	_, err = InstallQueue(rc, &butlerd.InstallQueueParams{
		CaveID: params.CaveID,
		Game:   cave.Game,
		Upload: cave.Upload,
		Build:  build,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := &butlerd.InstallVersionSwitchQueueResult{}
	return res, nil
}
