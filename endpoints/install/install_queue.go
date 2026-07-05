package install

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"crawshaw.io/sqlite"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/google/uuid"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/downloads"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/itchio/headway/state"
	"github.com/pkg/errors"
	"xorm.io/builder"
)

func InstallQueue(rc *butlerd.RequestContext, queueParams butlerd.InstallQueueParams) (*butlerd.InstallQueueResult, error) {
	var stagingFolder string
	conn := rc.GetConn()
	defer rc.PutConn(conn)

	var cave *models.Cave
	var installLocation *models.InstallLocation

	reason := queueParams.Reason
	if reason == "" {
		reason = butlerd.DownloadReasonInstall
	}

	var id string
	if queueParams.NoCave {
		if queueParams.StagingFolder == "" {
			return nil, errors.New("With noCave, StagingFolder must be specified")
		}
		stagingFolder = queueParams.StagingFolder
		id = uuid.New().String()
	} else {
		if queueParams.CaveID == "" {
			if queueParams.InstallLocationID == "" {
				return nil, errors.New("When caveId is unspecified, installLocationId must be set")
			}
			installLocation = models.InstallLocationByID(conn, queueParams.InstallLocationID)
			if installLocation == nil {
				return nil, errors.Errorf("Install location not found (%s)", queueParams.InstallLocationID)
			}
		} else {
			cave = operate.ValidateCave(rc, queueParams.CaveID)
			if queueParams.Game == nil {
				queueParams.Game = cave.Game
			}
			if queueParams.Upload == nil {
				queueParams.Upload = cave.Upload
			}
			if queueParams.Build == nil {
				queueParams.Build = cave.Build
			}

			installLocation = cave.GetInstallLocation(conn)
		}

		id = generateDownloadID(installLocation.Path)
		stagingFolder = installLocation.GetStagingFolder(id)
	}

	oc, err := operate.LoadContext(rc.Ctx, rc, stagingFolder)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	success := false
	defer func() {
		oc.Release()
		if !success {
			oc.Retire()
		}
	}()

	consumer := oc.Consumer()
	meta := operate.NewMetaSubcontext()
	params := meta.Data

	params.StagingFolder = stagingFolder
	params.Reason = reason
	params.IgnoreInstallers = queueParams.IgnoreInstallers

	if queueParams.Game == nil {
		return nil, errors.New("Missing game in install")
	}

	params.Game = queueParams.Game
	params.Access = operate.AccessForGameIDForProfile(conn, params.Game.ID, queueParams.ProfileID)

	client := rc.Client(params.Access.APIKey)

	// If ownership is via a bundle and no DownloadKey has been claimed yet,
	// materialize one now (before any upload listing or build resolution).
	// ClaimBundleGame is the single network call that converts bundle
	// ownership into a real DownloadKey row both server-side and locally.
	if err := ensureBundleAccessMaterialized(rc, conn, client, params.Game.ID, params.Access); err != nil {
		return nil, errors.WithStack(err)
	}

	consumer.Infof("Queuing install for %s", operate.GameToString(params.Game))

	freshCave := false
	if queueParams.NoCave {
		consumer.Infof("No-cave mode enabled")
		if queueParams.InstallFolder == "" {
			return nil, errors.New("With noCave is specified, InstallFolder cannot be empty")
		}

		params.NoCave = true
		params.InstallFolder = queueParams.InstallFolder
	} else {
		if cave == nil {
			freshCave = true
			cave = &models.Cave{
				ID:                uuid.New().String(),
				InstallLocationID: queueParams.InstallLocationID,
			}
			consumer.Infof("Generated fresh cave %s", cave.ID)
		} else {
			consumer.Infof("Re-using cave %s", cave.ID)
		}
		params.CaveID = cave.ID

		if cave.InstallFolderName == "" {
			cave.InstallFolderName = makeInstallFolderName(params.Game, consumer)
			ensureUniqueFolderName(conn, cave)
		}

		params.InstallFolder = cave.GetInstallFolder(conn)
		params.InstallLocationID = cave.InstallLocationID
		params.InstallFolderName = cave.InstallFolderName
	}

	params.Upload = queueParams.Upload
	params.Build = queueParams.Build

	if params.Upload == nil {
		consumer.Infof("No upload specified, looking for compatible ones...")
		uploadsFilterResult, err := operate.GetFilteredUploads(rc, params.Game)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if len(uploadsFilterResult.Uploads) == 0 {
			consumer.Errorf("Didn't find a compatible upload.")
			consumer.Errorf("The initial %d uploads were:", len(uploadsFilterResult.InitialUploads))
			for _, upload := range uploadsFilterResult.InitialUploads {
				operate.LogUpload(consumer, upload, upload.Build)
			}

			return nil, errors.WithStack(butlerd.CodeNoCompatibleUploads)
		}

		if len(uploadsFilterResult.Uploads) == 1 {
			params.Upload = uploadsFilterResult.Uploads[0]
		} else {
			r, err := messages.PickUpload.Call(rc, butlerd.PickUploadParams{
				Uploads: uploadsFilterResult.Uploads,
			})
			if err != nil {
				return nil, errors.WithStack(err)
			}

			if r.Index < 0 {
				return nil, errors.WithStack(butlerd.CodeOperationAborted)
			}

			params.Upload = uploadsFilterResult.Uploads[r.Index]
		}

		if params.Upload.Build != nil {
			// if we reach this point, we *just now* queried for an upload,
			// so we know the build object is the latest
			params.Build = params.Upload.Build
		}
	} else {
		consumer.Infof("Upload specified:")
		operate.LogUpload(consumer, params.Upload, params.Build)
	}

	if freshCave {
		dupCond := builder.Eq{
			"game_id":   params.Game.ID,
			"upload_id": params.Upload.ID,
		}
		if models.MustSelectOne(conn, &models.Cave{}, dupCond) {
			consumer.Errorf("Is fresh cave, and found a cave for game %d and upload %d, refusing to proceed.", params.Game.ID, params.Upload.ID)
			return nil, errors.Errorf("That upload is already installed!")
		}
	}

	// params.Upload can't be nil by now
	if params.Build == nil {
		// We were passed an upload but not a build:
		// Let's refresh upload info so we can settle on a build we want to install (if any)

		listUploadsRes, err := client.ListGameUploads(rc.Ctx, itchio.ListGameUploadsParams{
			GameID:      params.Game.ID,
			Credentials: params.Access.Credentials,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		found := true
		for _, u := range listUploadsRes.Uploads {
			if u.ID == params.Upload.ID {
				if u.Build == nil {
					consumer.Infof("Upload is not wharf-enabled")
				} else {
					consumer.Infof("Latest build for upload is %d", u.Build.ID)

					// now refresh the build itself, so we get a list of build files
					buildRes, err := client.GetBuild(rc.Ctx, itchio.GetBuildParams{
						BuildID:     u.Build.ID,
						Credentials: params.Access.Credentials,
					})
					if err != nil {
						return nil, errors.WithStack(err)
					}

					params.Build = buildRes.Build
				}
				break
			}
		}

		if !found {
			consumer.Errorf("Uh oh, we didn't find that upload on the server:")
			operate.LogUpload(consumer, params.Upload, nil)
			return nil, errors.New("Upload not found")
		}
	}

	oc.Save(meta)

	istate := &operate.InstallSubcontextState{}
	isub := &operate.InstallSubcontext{
		Data: istate,
	}
	oc.Load(isub)

	if queueParams.FastQueue {
		params.FastQueue = true
	} else {
		err = operate.InstallPrepare(oc, meta, isub, false /* disallow downloads */, func(res *operate.InstallPrepareResult) error {
			// do muffin
			return nil
		})
		if err != nil {
			oc.Retire()
			return nil, errors.WithStack(err)
		}

	}

	success = true
	res := &butlerd.InstallQueueResult{
		ID:                id,
		CaveID:            params.CaveID,
		Game:              params.Game,
		Upload:            params.Upload,
		Build:             params.Build,
		InstallFolder:     params.InstallFolder,
		StagingFolder:     params.StagingFolder,
		Reason:            params.Reason,
		InstallLocationID: params.InstallLocationID,
	}

	if queueParams.QueueDownload {
		_, err := downloads.DownloadsQueue(rc, butlerd.DownloadsQueueParams{
			Item: res,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return res, nil
}

// ensureBundleAccessMaterialized turns a deferred bundle ownership into a
// real DownloadKey by calling ClaimBundleGame and persisting the resulting
// key locally with owner_id set to access.ProfileID. Idempotent on the
// server. No-op when access is not bundle-owned or already has a
// DownloadKeyID.
func ensureBundleAccessMaterialized(rc *butlerd.RequestContext, conn *sqlite.Conn, client *itchio.Client, gameID int64, access *operate.GameAccess) error {
	if access == nil || access.BundleID == 0 || access.Credentials.DownloadKeyID != 0 {
		return nil
	}
	if access.ProfileID == 0 {
		return errors.New("cannot materialize bundle ownership without a profileId on access")
	}

	claimRes, err := client.ClaimBundleGame(rc.Ctx, itchio.ClaimBundleGameParams{
		BundleID: access.BundleID,
		GameID:   gameID,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	dk := claimRes.DownloadKey
	if dk == nil {
		return errors.New("ClaimBundleGame returned no download key")
	}

	fakeProfile := &models.Profile{ID: access.ProfileID}
	fakeProfile.OwnedKeys = []*itchio.DownloadKey{dk}
	models.MustSave(conn, fakeProfile,
		hades.OmitRoot(),
		hades.Assoc("OwnedKeys",
			hades.Assoc("Game"),
		),
	)

	// upload listings cached before the claim were fetched without owned
	// credentials (for paid games the server returns none); expire them so
	// the next read refetches with the new key
	models.FetchTargetForGameUploads(gameID).MustExpire(conn)

	access.Credentials.DownloadKeyID = dk.ID
	access.BundleID = 0
	return nil
}

// maybeMaterializeBundleAccess claims a download key for a game owned via a
// bundle, and is a no-op otherwise. Install-intent endpoints that list
// uploads (Install.GetUploads, Install.Plan, Game.FindUploads) call this
// first: the server does not list paid uploads without owned credentials,
// so an unclaimed bundle game would show no available downloads. Fetch-only
// endpoints (Fetch.GameUploads etc.) deliberately stay non-materializing.
//
// profileID scopes which profile the key is claimed against; zero falls
// back to any suitable profile.
func maybeMaterializeBundleAccess(rc *butlerd.RequestContext, conn *sqlite.Conn, gameID int64, profileID int64) error {
	access := operate.AccessForGameIDForProfile(conn, gameID, profileID)
	if access == nil || access.BundleID == 0 || access.Credentials.DownloadKeyID != 0 {
		return nil
	}
	client := rc.Client(access.APIKey)
	return ensureBundleAccessMaterialized(rc, conn, client, gameID, access)
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

func ensureUniqueFolderName(conn *sqlite.Conn, cave *models.Cave) {
	// Once we reach "Overland 200", it's time to stop
	const uniqueMaxTries = 200
	base := cave.InstallFolderName
	suffix := 2

	for i := 0; i < uniqueMaxTries; i++ {
		folder := cave.GetInstallFolder(conn)
		_, err := os.Stat(folder)
		alreadyExists := (err == nil)

		if !alreadyExists {
			// coolio
			return
		}

		// uh oh, it exists
		cave.InstallFolderName = fmt.Sprintf("%s %d", base, suffix)
		suffix++
	}

	cave.InstallFolderName = base
	err := errors.Errorf("Could not ensure unique install folder starting with (%s)", cave.GetInstallFolder(conn))
	panic(err)
}

func generateDownloadID(basePath string) string {
	for tries := 100; tries > 0; tries-- {
		id := petname.Generate(3, "-")
		_, err := os.Stat(filepath.Join(basePath, id))
		if err != nil && os.IsNotExist(err) {
			return id
		}
	}
	return uuid.New().String()
}
