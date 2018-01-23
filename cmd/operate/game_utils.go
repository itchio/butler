package operate

import (
	"encoding/json"
	"fmt"
	"strings"

	humanize "github.com/dustin/go-humanize"
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

	numInputs := len(uploads.Uploads)
	if numInputs == 0 {
		consumer.Infof("No uploads found at all (that we can access)")
	}
	uploadsFilterResult := manager.NarrowDownUploads(uploads.Uploads, game, manager.CurrentRuntime())

	numResults := len(uploadsFilterResult.Uploads)

	if numInputs > 0 {
		if numResults == 0 {
			consumer.Infof("→ All uploads were filtered out")
		}
		qualif := fmt.Sprintf("these %d uploads", numResults)
		if numResults == 1 {
			qualif = "this upload"
		}

		consumer.Infof("→ Narrowed %d uploads down to %s: ", numInputs, qualif)
		for _, u := range uploadsFilterResult.Uploads {
			logUpload(consumer, u, u.Build)
		}
	}

	return uploadsFilterResult, nil
}

func logUpload(consumer *state.Consumer, u *itchio.Upload, b *itchio.Build) {
	if u == nil {
		consumer.Infof("  No upload")
	} else {
		var name string
		if u.DisplayName != "" {
			name = u.DisplayName
		} else {
			name = u.Filename
		}

		var size string
		if u.Size > 0 {
			size = humanize.IBytes(uint64(u.Size))
		} else {
			size = "Unknown size"
		}

		consumer.Infof("  ☁ %s :: %s :: #%d", name, size, u.ID)

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

		var platString = "No platforms"
		if len(plats) > 0 {
			platString = strings.Join(plats, ", ")
		}

		consumer.Infof("    %s :: %s", formatUploadType(u.Type), platString)
	}

	if b != nil {
		version := ""
		if b.UserVersion != "" {
			version = fmt.Sprintf("%s", b.UserVersion)
		} else if b.Version != 0 {
			version = "No explicit version"
		}

		consumer.Infof("    Build %d for channel (%s) :: %s :: #%d", b.Version, u.ChannelName, version, b.ID)
	}
}

func formatUploadType(uploadType string) string {
	switch uploadType {
	case "default":
		return "Executable"

	case "flash":
		return "Flash object"
	case "unity":
		return "Legacy Unity Web"
	case "java":
		return "Java applet"

	case "soundtrack":
		return "Soundtrack"
	case "book":
		return "Book"
	case "video":
		return "Video"
	case "documentation":
		return "Documentation"
	case "mod":
		return "Mod"
	case "audio_assets":
		return "Audio assets"
	case "graphical_assets":
		return "Graphical assets"
	case "sourcecode":
		return "Source code"

	case "other":
		return "Other"

	default:
		return fmt.Sprintf("(%s)", uploadType)
	}
}
