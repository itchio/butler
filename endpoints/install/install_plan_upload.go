package install

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	itchio "github.com/itchio/go-itchio"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"

	"github.com/itchio/hades"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"

	"github.com/itchio/hush"
	"github.com/itchio/hush/bfs"

	"github.com/pkg/errors"
	"xorm.io/builder"
)

func InstallPlanUpload(rc *butlerd.RequestContext, params butlerd.InstallPlanUploadParams) (*butlerd.InstallPlanUploadResult, error) {
	consumer := rc.Consumer

	if params.ID != "" {
		_, cleanup := rc.MakeCancelable(params.ID)
		defer cleanup()
	}

	conn := rc.GetConn()
	defer rc.PutConn(conn)

	info := &butlerd.InstallPlanInfo{}
	res := &butlerd.InstallPlanUploadResult{
		Info: info,
	}

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

	var upload itchio.Upload
	if ok := models.MustSelectOne(conn, &upload, builder.Eq{"id": params.UploadID}); !ok {
		return nil, errors.Errorf("upload %d not found", params.UploadID)
	}
	models.MustPreload(conn, &upload, hades.Assoc("Build"))

	var gameUpload models.GameUpload
	if ok := models.MustSelectOne(conn, &gameUpload, builder.Eq{"upload_id": params.UploadID}); !ok {
		return nil, errors.Errorf("game upload mapping not found for upload %d", params.UploadID)
	}

	access := operate.AccessForGameID(conn, gameUpload.GameID)
	client := rc.Client(access.APIKey)

	info.Upload = &upload
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
	operate.LogUpload(consumer, &upload, upload.Build)

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

	if err := checkCancelled(rc.Ctx); err != nil {
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

	if err := checkCancelled(rc.Ctx); err != nil {
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
