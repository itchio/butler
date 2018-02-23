package install

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/operate"
)

func GameFindUploads(rc *buse.RequestContext, params *buse.GameFindUploadsParams) (*buse.GameFindUploadsResult, error) {
	consumer := rc.Consumer

	if params.Game == nil {
		return nil, errors.New("Missing game")
	}

	consumer.Infof("Looking for compatible uploads for game %s", operate.GameToString(params.Game))

	client, err := rc.Harness.ClientFromCredentials(params.Credentials)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	uploads, err := operate.GetFilteredUploads(client, params.Game, params.Credentials, consumer)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.GameFindUploadsResult{
		Uploads: uploads.Uploads,
	}
	return res, nil
}
