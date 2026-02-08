package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

func FetchUploadBuilds(rc *butlerd.RequestContext, params butlerd.FetchUploadBuildsParams) (*butlerd.FetchUploadBuildsResult, error) {
	res := &butlerd.FetchUploadBuildsResult{}

	var access *operate.GameAccess
	rc.WithConn(func(conn *sqlite.Conn) {
		access = operate.AccessForGameID(conn, params.Game.ID)
	})
	client := rc.Client(access.APIKey)

	buildsRes, err := client.ListUploadBuilds(rc.Ctx, itchio.ListUploadBuildsParams{
		UploadID:    params.Upload.ID,
		Credentials: access.Credentials,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res.Builds = buildsRes.Builds

	return res, nil
}
