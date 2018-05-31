package install

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/pkg/errors"
)

func GameFindUploads(rc *butlerd.RequestContext, params *butlerd.GameFindUploadsParams) (*butlerd.GameFindUploadsResult, error) {
	consumer := rc.Consumer

	if params.Game == nil {
		return nil, errors.New("Missing game")
	}

	consumer.Infof("Looking for compatible uploads for game %s", operate.GameToString(params.Game))

	access := operate.AccessForGameID(rc.DB(), params.Game.ID)
	client := rc.Client(access.APIKey)

	uploads, err := operate.GetFilteredUploads(client, params.Game, access.Credentials, consumer)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := &butlerd.GameFindUploadsResult{
		Uploads: uploads.Uploads,
	}
	return res, nil
}
