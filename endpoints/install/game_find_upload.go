package install

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/pkg/errors"
)

func GameFindUploads(rc *butlerd.RequestContext, params butlerd.GameFindUploadsParams) (*butlerd.GameFindUploadsResult, error) {
	consumer := rc.Consumer

	if params.Game == nil {
		return nil, errors.New("Missing game")
	}

	consumer.Infof("Looking for compatible uploads for game %s", operate.GameToString(params.Game))

	// install intent: claim bundle-owned games before listing uploads
	var materializeErr error
	rc.WithConn(func(conn *sqlite.Conn) {
		materializeErr = maybeMaterializeBundleAccess(rc, conn, params.Game.ID)
	})
	if materializeErr != nil {
		return nil, errors.WithStack(materializeErr)
	}

	uploads, err := operate.GetFilteredUploads(rc, params.Game)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := &butlerd.GameFindUploadsResult{
		Uploads: uploads.Uploads,
	}
	return res, nil
}
