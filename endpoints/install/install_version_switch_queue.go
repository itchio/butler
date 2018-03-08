package install

import (
	"fmt"

	"github.com/itchio/butler/buse/messages"
	itchio "github.com/itchio/go-itchio"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/operate"
)

func InstallVersionSwitchQueue(rc *buse.RequestContext, params *buse.InstallVersionSwitchQueueParams) (*buse.InstallVersionSwitchQueueResult, error) {
	consumer := rc.Consumer

	cave := operate.ValidateCave(rc, params.CaveID)

	consumer.Infof("Looking for other versions of %s", operate.GameToString(cave.Game))

	upload := cave.Upload
	if upload == nil {
		return nil, fmt.Errorf("No other versions available for %s", operate.GameToString(cave.Game))
	}

	credentials := operate.CredentialsForGame(rc.DB(), consumer, cave.Game)

	client, err := operate.ClientFromCredentials(credentials)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	buildsRes, err := client.ListUploadBuilds(&itchio.ListUploadBuildsParams{
		UploadID:      upload.ID,
		DownloadKeyID: credentials.DownloadKey,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	pickRes, err := messages.InstallVersionSwitchPick.Call(rc, &buse.InstallVersionSwitchPickParams{
		Upload: upload,
		Builds: buildsRes.Builds,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if pickRes.Index < 0 {
		return nil, &buse.ErrAborted{}
	}

	build := buildsRes.Builds[pickRes.Index]

	if true {
		return nil, errors.New("We're up to InstallQueue and butler doesn't fully handle downloads yet :o")
	}

	_, err = InstallQueue(rc, &buse.InstallQueueParams{
		CaveID: params.CaveID,
		Game:   cave.Game,
		Upload: cave.Upload,
		Build:  build,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.InstallVersionSwitchQueueResult{}
	return res, nil
}
