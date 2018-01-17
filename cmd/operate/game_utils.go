package operate

import (
	"encoding/json"
	"fmt"
	"strings"

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
	for _, u := range uploadsFilterResult.Uploads {
		logUpload(consumer, u, u.Build)
	}

	return uploadsFilterResult, nil
}

func logUpload(consumer *state.Consumer, u *itchio.Upload, b *itchio.Build) {
	if u == nil {
		consumer.Infof("  No upload")
	} else {
		consumer.Infof("  Upload %d (%s): %s", u.ID, u.Filename, u.DisplayName)

		ch := "No channel"
		if u.ChannelName != "" {
			ch = fmt.Sprintf("Channel '%s'", u.ChannelName)
		}

		var plats []string
		if u.Linux {
			plats = append(plats, "Linux")
		}
		if u.Windows {
			plats = append(plats, "Windows")
		}
		if u.OSX {
			plats = append(plats, "macOS")
		}
		if u.Android {
			plats = append(plats, "Android")
		}

		var platString = "(none)"
		if len(plats) > 0 {
			platString = strings.Join(plats, ", ")
		}

		consumer.Infof("  ...%s, Type: %s, Platforms: %s", ch, u.Type, platString)
	}

	if b != nil {
		additionalInfo := ""
		if b.UserVersion != "" {
			additionalInfo = fmt.Sprintf(", version %s", b.UserVersion)
		} else if b.Version != 0 {
			additionalInfo = fmt.Sprintf(", number %d", b.Version)
		}

		consumer.Infof("  ...Build %d%s", b.ID, additionalInfo)
	}
}
