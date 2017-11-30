package operate

import (
	"encoding/json"
	"fmt"

	"github.com/itchio/butler/manager"
	"github.com/itchio/wharf/state"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	itchio "github.com/itchio/go-itchio"
	"github.com/sourcegraph/jsonrpc2"
)

func GameFindUploads(ctx *mansion.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (*buse.GameFindUploadsResult, error) {
	consumer := comm.NewStateConsumer()

	params := &buse.GameFindUploadsParams{}
	err := json.Unmarshal(*req.Params, params)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if params.Game == nil {
		return nil, errors.New("Missing game")
	}

	consumer.Infof("Looking for compatible uploads for game %s", gameToString(params.Game))

	client, err := clientFromCredentials(params.Credentials)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	uploads, err := getFilteredUploads(client, params.Game, params.Credentials, consumer)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.GameFindUploadsResult{
		Uploads: uploads.Uploads,
	}
	return res, nil
}

func gameToString(game *itchio.Game) string {
	if game == nil {
		return "<nil game>"
	}

	return fmt.Sprintf("%s - %s", game.Title, game.URL)
}

func getFilteredUploads(client *itchio.Client, game *itchio.Game, credentials *buse.GameCredentials, consumer *state.Consumer) (*manager.NarrowDownUploadsResult, error) {
	uploads, err := client.ListGameUploads(&itchio.ListGameUploadsParams{
		GameID:        game.ID,
		DownloadKeyID: credentials.DownloadKey,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Infof("Filtering %d uploads", len(uploads.Uploads))
	uploadsFilterResult := manager.NarrowDownUploads(uploads.Uploads, game, manager.CurrentRuntime())
	consumer.Infof("After filter, got %d uploads, they are: ", len(uploadsFilterResult.Uploads))
	for _, upload := range uploadsFilterResult.Uploads {
		consumer.Infof("- %#v", upload)
	}

	return uploadsFilterResult, nil
}
