package install

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
)

func GameFindUploads(rc *butlerd.RequestContext, params *butlerd.GameFindUploadsParams) (*butlerd.GameFindUploadsResult, error) {
	consumer := rc.Consumer

	if params.Game == nil {
		return nil, errors.New("Missing game")
	}

	consumer.Infof("Looking for compatible uploads for game %s", operate.GameToString(params.Game))

	credentials := operate.CredentialsForGameID(rc.DB(), params.Game.ID)
	client, err := operate.ClientFromCredentials(credentials)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	uploads, err := operate.GetFilteredUploads(client, params.Game, credentials, consumer)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &butlerd.GameFindUploadsResult{
		Uploads: uploads.Uploads,
	}
	return res, nil
}
