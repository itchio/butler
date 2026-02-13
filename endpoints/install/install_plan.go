package install

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

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

func InstallPlan(rc *butlerd.RequestContext, params butlerd.InstallPlanParams) (*butlerd.InstallPlanResult, error) {
	consumer := rc.Consumer

	ctx := rc.Ctx
	if params.ID != "" {
		var cancelFunc context.CancelFunc
		ctx, cancelFunc = context.WithCancel(ctx)
		rc.CancelFuncs.Add(params.ID, cancelFunc)
		defer rc.CancelFuncs.Remove(params.ID)
	}

	conn := rc.GetConn()
	defer rc.PutConn(conn)

	game := fetch.LazyFetchGame(rc, params.GameID)
	if err := checkCancelled(ctx); err != nil {
		return nil, err
	}
	consumer.Opf("Planning install for %s", operate.GameToString(game))

	baseUploads := fetch.LazyFetchGameUploads(rc, params.GameID)
	if err := checkCancelled(ctx); err != nil {
		return nil, err
	}

	narrowRes, err := manager.NarrowDownUploads(consumer, game, baseUploads, rc.HostEnumerator())
	if err != nil {
		return nil, err
	}
	baseUploads = narrowRes.Uploads

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
	// do a little dance to keep the ordering proper
	var uploads []*itchio.Upload
	for _, u := range baseUploads {
		if validUploadIDs[u.ID] {
			uploads = append(uploads, u)
		}
	}

	res := &butlerd.InstallPlanResult{
		Game:    game,
		Uploads: uploads,
	}

	if len(uploads) == 0 {
		consumer.Statf("No compatible uploads, returning early.")
		return res, nil
	}

	info := &butlerd.InstallPlanInfo{}
	res.Info = info

	setResError := func(err error) {
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

	var upload *itchio.Upload
	if params.UploadID != 0 {
		for _, u := range uploads {
			if u.ID == params.UploadID {
				consumer.Infof("Using specified upload.")
				upload = u
				break
			}
		}
	}

	if upload == nil {
		consumer.Infof("Picking first upload.")
		upload = uploads[0]
	}

	access := operate.AccessForGameID(conn, game.ID)
	client := rc.Client(access.APIKey)

	info.Upload = upload
	if upload.Build != nil {
		buildRes, err := client.GetBuild(ctx, itchio.GetBuildParams{
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
		setResError(errors.WithStack(butlerd.CodeUnsupportedHost))
		return res, nil
	}

	sessionID := params.DownloadSessionID
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

	if err := checkCancelled(ctx); err != nil {
		return nil, err
	}

	beforeOpen := time.Now()
	file, err := eos.Open(sourceURL, option.WithConsumer(consumer))
	consumer.Infof("(opening file took %s)", time.Since(beforeOpen))
	if err != nil {
		setResError(errors.WithStack(err))
		return res, nil
	}
	defer file.Close()

	if err := checkCancelled(ctx); err != nil {
		return nil, err
	}

	installerInfo, err := hush.GetInstallerInfo(consumer, file)
	if err != nil {
		setResError(errors.WithStack(err))
		return res, nil
	}

	info.Type = string(installerInfo.Type)

	// planning is always for a fresh install
	receiptIn := (*bfs.Receipt)(nil)
	installFolder := ""

	dui, err := operate.AssessDiskUsage(file, receiptIn, installFolder, installerInfo)
	if err != nil {
		setResError(errors.WithStack(err))
		return res, nil
	}

	info.DiskUsage = &butlerd.DiskUsageInfo{
		FinalDiskUsage:  dui.FinalDiskUsage,
		NeededFreeSpace: dui.NeededFreeSpace,
		Accuracy:        dui.Accuracy.String(),
	}

	return res, nil
}
