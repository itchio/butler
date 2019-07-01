package install

import (
	"fmt"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/endpoints/fetch"
	itchio "github.com/itchio/go-itchio"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/pkg/errors"
)

func InstallVersionSwitchQueue(rc *butlerd.RequestContext, params butlerd.InstallVersionSwitchQueueParams) (*butlerd.InstallVersionSwitchQueueResult, error) {
	consumer := rc.Consumer

	cave := operate.ValidateCave(rc, params.CaveID)

	consumer.Infof("Looking for other versions of %s", operate.GameToString(cave.Game))

	upload := cave.Upload
	if upload == nil {
		return nil, fmt.Errorf("No other versions available for %s", operate.GameToString(cave.Game))
	}

	var access *operate.GameAccess
	rc.WithConn(func(conn *sqlite.Conn) {
		access = operate.AccessForGameID(conn, cave.Game.ID)
	})
	client := rc.Client(access.APIKey)

	buildsRes, err := client.ListUploadBuilds(rc.Ctx, itchio.ListUploadBuildsParams{
		UploadID:    upload.ID,
		Credentials: access.Credentials,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var formattedCave *butlerd.Cave
	rc.WithConn(func(conn *sqlite.Conn) {
		formattedCave = fetch.FormatCave(conn, cave)
	})

	pickRes, err := messages.InstallVersionSwitchPick.Call(rc, butlerd.InstallVersionSwitchPickParams{
		Cave:   formattedCave,
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

	_, err = InstallQueue(rc, butlerd.InstallQueueParams{
		CaveID:        params.CaveID,
		Game:          cave.Game,
		Upload:        cave.Upload,
		Build:         build,
		Reason:        butlerd.DownloadReasonVersionSwitch,
		QueueDownload: true,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := &butlerd.InstallVersionSwitchQueueResult{}
	return res, nil
}
