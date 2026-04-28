package install

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"crawshaw.io/sqlite"

	itchio "github.com/itchio/go-itchio"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch"
	"github.com/itchio/butler/manager"

	"github.com/itchio/hades"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"

	"github.com/itchio/hush"
	"github.com/itchio/hush/bfs"

	"github.com/pkg/errors"
	"xorm.io/builder"
)

func checkCancelled(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return errors.WithStack(butlerd.CodeOperationCancelled)
	default:
		return nil
	}
}

// getGameUploads fetches the game and its uploads, narrows by platform/format,
// and excludes already-installed or in-progress uploads.
func getGameUploads(rc *butlerd.RequestContext, conn *sqlite.Conn, gameID int64) (*itchio.Game, []*itchio.Upload, error) {
	consumer := rc.Consumer

	game := fetch.LazyFetchGame(rc, gameID)
	if err := checkCancelled(rc.Ctx); err != nil {
		return nil, nil, err
	}
	consumer.Opf("Planning install for %s", operate.GameToString(game))

	baseUploads := fetch.LazyFetchGameUploads(rc, gameID)
	if err := checkCancelled(rc.Ctx); err != nil {
		return nil, nil, err
	}

	narrowRes, err := manager.NarrowDownUploads(consumer, game, baseUploads, rc.HostEnumerator())
	if err != nil {
		return nil, nil, err
	}

	if len(narrowRes.Uploads) != 0 {
		consumer.Statf("No compatible uploads, showing incompatible uploads as well.")
		baseUploads = narrowRes.Uploads
	}

	// exclude already-installed and currently-installing uploads
	var uploadIDs []interface{}
	for _, u := range baseUploads {
		uploadIDs = append(uploadIDs, u.ID)
	}
	var validUploads []*itchio.Upload
	models.MustSelect(conn, &validUploads, builder.And(
		builder.In("id", uploadIDs...),
		builder.Expr(`not exists (select 1 from caves where upload_id = uploads.id)`),
		builder.Expr(`not exists (select 1 from downloads where upload_id = uploads.id and finished_at is null and not discarded)`),
	), hades.Search{})
	validUploadIDs := make(map[int64]bool)
	for _, u := range validUploads {
		validUploadIDs[u.ID] = true
	}
	// keep the ordering proper
	var uploads []*itchio.Upload
	for _, u := range baseUploads {
		if validUploadIDs[u.ID] {
			uploads = append(uploads, u)
		}
	}

	return game, uploads, nil
}

// getPlanInfo resolves build info, opens the remote file, gets installer info,
// and assesses disk usage for a single upload. Errors are stored in the returned
// InstallPlanInfo rather than returned, matching the original behavior where
// planning errors are soft failures.
func getPlanInfo(rc *butlerd.RequestContext, conn *sqlite.Conn, upload *itchio.Upload, gameID int64, downloadSessionID string) (*butlerd.InstallPlanInfo, error) {
	consumer := rc.Consumer

	info := &butlerd.InstallPlanInfo{}

	setInfoError := func(err error) {
		consumer.Errorf("Planning failed: %+v", err)
		info.Error = fmt.Sprintf("%+v", err)
		if be, ok := butlerd.AsButlerdError(err); ok {
			info.ErrorCode = be.RpcErrorCode()
			info.ErrorMessage = be.RpcErrorMessage()
		} else {
			info.ErrorCode = int64(jsonrpc2.CodeInternalError)
			info.ErrorMessage = err.Error()
		}
	}

	access := operate.AccessForGameID(conn, gameID)
	client := rc.Client(access.APIKey)

	info.Upload = upload
	if upload.Build != nil {
		buildRes, err := client.GetBuild(rc.Ctx, itchio.GetBuildParams{
			BuildID:     upload.Build.ID,
			Credentials: access.Credentials,
		})
		if err != nil {
			return nil, err
		}

		upload.Build = buildRes.Build
	}
	info.Build = upload.Build
	operate.LogUpload(consumer, upload, upload.Build)

	if upload.Storage == itchio.UploadStorageExternal && operate.IsBadExternalHost(upload.Host) {
		setInfoError(errors.WithStack(butlerd.CodeUnsupportedHost))
		return info, nil
	}

	sessionID := downloadSessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
		consumer.Infof("No download session ID passed, using %s", sessionID)
	}

	installParams := &operate.InstallParams{
		Upload: info.Upload,
		Build:  info.Build,
		Access: access,
	}
	sourceURL := operate.MakeSourceURL(client, consumer, sessionID, installParams, "")

	if err := checkCancelled(rc.Ctx); err != nil {
		return nil, err
	}

	beforeOpen := time.Now()
	file, err := eos.Open(sourceURL, option.WithConsumer(consumer))
	consumer.Infof("(opening file took %s)", time.Since(beforeOpen))
	if err != nil {
		setInfoError(errors.WithStack(err))
		return info, nil
	}
	defer file.Close()

	if err := checkCancelled(rc.Ctx); err != nil {
		return nil, err
	}

	installerInfo, err := hush.GetInstallerInfo(consumer, file)
	if err != nil {
		setInfoError(errors.WithStack(err))
		return info, nil
	}

	info.Type = string(installerInfo.Type)

	// planning is always for a fresh install
	receiptIn := (*bfs.Receipt)(nil)
	installFolder := ""

	dui, err := operate.AssessDiskUsage(file, receiptIn, installFolder, installerInfo)
	if err != nil {
		setInfoError(errors.WithStack(err))
		return info, nil
	}

	info.DiskUsage = &butlerd.DiskUsageInfo{
		FinalDiskUsage:  dui.FinalDiskUsage,
		NeededFreeSpace: dui.NeededFreeSpace,
		Accuracy:        dui.Accuracy.String(),
	}

	return info, nil
}

// InstallPlan is the deprecated handler that returns Game, Uploads, and Info.
func InstallPlan(rc *butlerd.RequestContext, params butlerd.InstallPlanParams) (*butlerd.InstallPlanResult, error) {
	conn := rc.GetConn()
	defer rc.PutConn(conn)

	game, uploads, err := getGameUploads(rc, conn, params.GameID)
	if err != nil {
		return nil, err
	}

	res := &butlerd.InstallPlanResult{
		Game:    game,
		Uploads: uploads,
	}

	// Select the upload to plan: explicit UploadID or first available
	var selectedUpload *itchio.Upload
	if params.UploadID != 0 {
		for _, u := range uploads {
			if u.ID == params.UploadID {
				selectedUpload = u
				break
			}
		}
	} else if len(uploads) > 0 {
		selectedUpload = uploads[0]
	}

	if selectedUpload != nil {
		info, err := getPlanInfo(rc, conn, selectedUpload, params.GameID, params.DownloadSessionID)
		if err != nil {
			return nil, err
		}
		res.Info = info
	}

	return res, nil
}

// InstallGetUploads returns the game and available uploads (fast path, no file I/O).
func InstallGetUploads(rc *butlerd.RequestContext, params butlerd.InstallGetUploadsParams) (*butlerd.InstallGetUploadsResult, error) {
	conn := rc.GetConn()
	defer rc.PutConn(conn)

	game, uploads, err := getGameUploads(rc, conn, params.GameID)
	if err != nil {
		return nil, err
	}

	return &butlerd.InstallGetUploadsResult{
		Game:    game,
		Uploads: uploads,
	}, nil
}
