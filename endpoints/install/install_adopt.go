package install

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"crawshaw.io/sqlite"
	"github.com/google/uuid"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch"
	"github.com/itchio/butler/manager"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/itchio/hush/bfs"
	"github.com/itchio/ox"
	"github.com/pkg/errors"
	"xorm.io/builder"
)

// InstallAdopt registers a ready-to-run folder that already lives directly
// beneath a configured install location. The adoption operation itself never
// copies or modifies the folder's game files; after adoption, the entire folder
// is managed by butler and will be deleted when the cave is uninstalled.
func InstallAdopt(rc *butlerd.RequestContext, params butlerd.InstallAdoptParams) (*butlerd.InstallAdoptResult, error) {
	conn := rc.GetConn()
	defer rc.PutConn(conn)

	consumer := rc.Consumer
	installLocation := models.InstallLocationByID(conn, params.InstallLocationID)
	if installLocation == nil {
		return nil, errors.Errorf("Install location not found (%s)", params.InstallLocationID)
	}

	if err := validateAdoptFolderName(params.InstallFolderName); err != nil {
		return nil, err
	}

	installFolder := installLocation.GetInstallFolder(params.InstallFolderName)
	folderInfo, err := os.Lstat(installFolder)
	if err != nil {
		return nil, errors.Wrap(err, "statting folder to adopt")
	}
	if folderInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.Errorf("Refusing to adopt symlinked folder (%s)", installFolder)
	}
	if !folderInfo.IsDir() {
		return nil, errors.Errorf("Folder to adopt is not a directory (%s)", installFolder)
	}

	receiptPath := bfs.ReceiptPath(installFolder)
	if _, err := os.Stat(receiptPath); err == nil {
		return nil, errors.Errorf("Folder already has a receipt; scan install locations instead (%s)", installFolder)
	} else if !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "checking for an existing receipt")
	}

	if caveAtLocation(conn, params.InstallLocationID, params.InstallFolderName) {
		return nil, errors.Errorf("Folder is already registered as an installed item (%s)", installFolder)
	}
	if caveForGameUpload(conn, params.GameID, params.UploadID) {
		return nil, errors.Errorf("Upload %d is already installed", params.UploadID)
	}
	if activeDownloadForUploadOrFolder(conn, params.UploadID, installFolder) {
		return nil, errors.Errorf("An active download already targets upload %d or folder (%s)", params.UploadID, installFolder)
	}

	if err := maybeMaterializeBundleAccess(rc, conn, params.GameID, params.ProfileID); err != nil {
		return nil, errors.WithStack(err)
	}

	game := fetch.LazyFetchGame(rc, params.GameID)
	var upload *itchio.Upload
	for _, candidate := range fetch.LazyFetchGameUploads(rc, params.GameID) {
		if candidate.ID == params.UploadID {
			upload = candidate
			break
		}
	}
	if upload == nil {
		return nil, errors.Errorf("Upload %d does not belong to game %d or is not accessible", params.UploadID, params.GameID)
	}

	access := operate.AccessForGameIDForProfile(conn, params.GameID, params.ProfileID)
	client := rc.Client(access.APIKey)
	var build *itchio.Build
	if params.BuildID != 0 {
		buildRes, err := client.GetBuild(rc.Ctx, itchio.GetBuildParams{
			BuildID:     params.BuildID,
			Credentials: access.Credentials,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}
		build = buildRes.Build
		// Upload IDs are globally unique and the upload was already verified to
		// belong to the requested game. Some API implementations omit GameID on
		// a standalone build response, so only validate it when present.
		if build == nil || build.UploadID != upload.ID || (build.GameID != 0 && build.GameID != game.ID) {
			return nil, errors.Errorf("Build %d does not belong to upload %d and game %d", params.BuildID, params.UploadID, params.GameID)
		}
	} else if upload.Build != nil {
		buildRes, err := client.GetBuild(rc.Ctx, itchio.GetBuildParams{
			BuildID:     upload.Build.ID,
			Credentials: access.Credentials,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}
		build = buildRes.Build
	}

	consumer.Opf("Configuring adopted folder for %s", ox.CurrentRuntime())
	verdict, err := manager.Configure(consumer, installFolder, ox.CurrentRuntime())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	installedAt := time.Now().UTC()
	cave := &models.Cave{
		ID:                uuid.New().String(),
		Game:              game,
		Upload:            upload,
		Build:             build,
		InstalledAt:       &installedAt,
		InstallLocationID: params.InstallLocationID,
		InstallFolderName: params.InstallFolderName,
		InstalledSize:     verdict.TotalSize,
	}
	cave.SetVerdict(verdict)

	err = models.HadesContext().Save(conn, cave,
		hades.Assoc("Game"),
		hades.Assoc("Upload"),
		hades.Assoc("Build"),
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	receipt := &bfs.Receipt{
		Game:          game,
		Upload:        upload,
		Build:         build,
		Files:         []string{},
		InstallerName: "",
	}
	if err := receipt.WriteReceipt(installFolder); err != nil {
		// The user's files predate this operation and must never be touched on
		// rollback. Remove only metadata created by this operation.
		cave.Delete(conn)
		_ = os.Remove(receiptPath)
		// Remove .itch only if this operation created it and it is still empty.
		_ = os.Remove(filepath.Dir(receiptPath))
		return nil, errors.WithStack(err)
	}

	cave.Preload(conn)
	return &butlerd.InstallAdoptResult{Cave: fetch.FormatCave(conn, cave)}, nil
}

func validateAdoptFolderName(name string) error {
	if name == "." || name == ".." || filepath.IsAbs(name) || filepath.Clean(name) != name || strings.ContainsAny(name, `/\\`) {
		return errors.Errorf("installFolderName must be a single folder name")
	}
	if strings.EqualFold(name, "downloads") {
		return errors.Errorf("cannot adopt the reserved downloads folder")
	}
	return nil
}

func caveAtLocation(conn *sqlite.Conn, installLocationID string, installFolderName string) bool {
	return models.MustCount(conn, &models.Cave{}, builder.Eq{
		"install_location_id": installLocationID,
		"install_folder_name": installFolderName,
	}) > 0
}

func caveForGameUpload(conn *sqlite.Conn, gameID int64, uploadID int64) bool {
	return models.MustCount(conn, &models.Cave{}, builder.Eq{
		"game_id":   gameID,
		"upload_id": uploadID,
	}) > 0
}

func activeDownloadForUploadOrFolder(conn *sqlite.Conn, uploadID int64, installFolder string) bool {
	return models.MustCount(conn, &models.Download{}, builder.And(
		builder.IsNull{"finished_at"},
		builder.Eq{"discarded": false},
		builder.Or(
			builder.Eq{"upload_id": uploadID},
			builder.Eq{"install_folder": installFolder},
		),
	)) > 0
}
