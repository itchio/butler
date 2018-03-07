package operate

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	"github.com/itchio/butler/database/models"
	"github.com/itchio/wharf/state"
	uuid "github.com/satori/go.uuid"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	itchio "github.com/itchio/go-itchio"
)

func InstallQueue(ctx context.Context, rc *buse.RequestContext, queueParams *buse.InstallQueueParams) error {
	if queueParams.StagingFolder == "" {
		return errors.New("No staging folder specified")
	}

	oc, err := LoadContext(ctx, rc, queueParams.StagingFolder)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer := oc.Consumer()
	meta := NewMetaSubcontext()
	params := meta.data

	params.StagingFolder = queueParams.StagingFolder

	params.Game = queueParams.Game
	if params.Game == nil {
		return errors.New("Missing game in install")
	}

	db, err := rc.DB()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if queueParams.NoCave {
		if queueParams.InstallFolder == "" {
			return errors.New("When NoCave is specified, InstallFolder cannot be empty")
		}

		params.NoCave = true
		params.InstallFolder = queueParams.InstallFolder
	} else {
		caveID, err := uuid.NewV4()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		params.CaveID = caveID.String()

		var installLocationName string
		var installFolderName string
		if queueParams.CaveID != "" {
			cave, err := models.CaveByID(db, queueParams.CaveID)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			if cave == nil {
				return fmt.Errorf("Cave not found: (%s)", queueParams.CaveID)
			}
			params.CaveID = cave.ID
			installLocationName = cave.InstallLocation
			installFolderName = cave.InstallFolder
		} else {
			if queueParams.InstallLocation == "" {
				return errors.New("Must specify either CaveID or InstallLocation")
			}
			installLocationName = queueParams.InstallLocation
		}

		installLocation, err := models.InstallLocationByID(db, installLocationName)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if installLocation == nil {
			return fmt.Errorf("Install location not found: (%s)", installLocationName)
		}

		if installFolderName == "" {
			installFolderName = makeInstallFolderName(params.Game, consumer)
		}
		params.InstallFolder = installLocation.AbsoluteFolderPath(installFolderName)
		params.InstallLocationName = installLocationName
		params.InstallFolderName = installFolderName
	}

	params.Credentials = queueParams.Credentials
	if params.Credentials == nil {
		credentials, err := CredentialsForGame(db, consumer, params.Game)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		params.Credentials = credentials
	}

	client, err := ClientFromCredentials(params.Credentials)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	params.Upload = queueParams.Upload
	params.Build = queueParams.Build

	if params.Upload == nil {
		consumer.Infof("No upload specified, looking for compatible ones...")
		uploadsFilterResult, err := GetFilteredUploads(client, params.Game, params.Credentials, consumer)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if len(uploadsFilterResult.Uploads) == 0 {
			consumer.Errorf("Didn't find a compatible upload.")
			consumer.Errorf("The initial %d uploads were:", len(uploadsFilterResult.InitialUploads))
			for _, upload := range uploadsFilterResult.InitialUploads {
				LogUpload(consumer, upload, upload.Build)
			}

			return (&OperationError{
				Code:      "noCompatibleUploads",
				Message:   "No compatible uploads",
				Operation: "install",
			}).Throw()
		}

		if len(uploadsFilterResult.Uploads) == 1 {
			params.Upload = uploadsFilterResult.Uploads[0]
		} else {
			r, err := messages.PickUpload.Call(oc.rc, &buse.PickUploadParams{
				Uploads: uploadsFilterResult.Uploads,
			})
			if err != nil {
				return errors.Wrap(err, 0)
			}

			if r.Index < 0 {
				return &buse.ErrAborted{}
			}

			params.Upload = uploadsFilterResult.Uploads[r.Index]
		}

		if params.Upload.Build != nil {
			// if we reach this point, we *just now* queried for an upload,
			// so we know the build object is the latest
			params.Build = params.Upload.Build
		}
	}

	// params.Upload can't be nil by now
	if params.Build == nil {
		// We were passed an upload but not a build:
		// Let's refresh upload info so we can settle on a build we want to install (if any)

		listUploadsRes, err := client.ListGameUploads(&itchio.ListGameUploadsParams{
			GameID:        params.Game.ID,
			DownloadKeyID: params.Credentials.DownloadKey,
		})
		if err != nil {
			return errors.Wrap(err, 0)
		}

		found := true
		for _, u := range listUploadsRes.Uploads {
			if u.ID == params.Upload.ID {
				if u.Build == nil {
					consumer.Infof("Upload is not wharf-enabled")
				} else {
					consumer.Infof("Latest build for upload is %d", u.Build.ID)
					params.Build = u.Build
				}
				break
			}
		}

		if !found {
			consumer.Errorf("Uh oh, we didn't find that upload on the server:")
			LogUpload(consumer, params.Upload, nil)
			return errors.New("Upload not found")
		}
	}

	oc.Save(meta)
	return nil
}

func makeInstallFolderName(game *itchio.Game, consumer *state.Consumer) string {
	name := makeInstallFolderNameFromSlug(game, consumer)
	if name == "" {
		name = makeInstallFolderNameFromID(game, consumer)
	}
	return name
}

var slugRe = regexp.MustCompile(`^\/([^\/]+)`)

func makeInstallFolderNameFromSlug(game *itchio.Game, consumer *state.Consumer) string {
	if game.URL == "" {
		return ""
	}

	u, err := url.Parse(game.URL)
	if err != nil {
		consumer.Warnf("Could not parse game URL (%s): %s", game.URL, err.Error())
		return ""
	}

	matches := slugRe.FindStringSubmatch(u.Path)
	if len(matches) == 2 {
		return matches[1]
	}

	return ""
}

func makeInstallFolderNameFromID(game *itchio.Game, consumer *state.Consumer) string {
	return fmt.Sprintf("game-%d", game.ID)
}
