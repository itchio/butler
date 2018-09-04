package install

import (
	"github.com/go-xorm/builder"
	"github.com/google/uuid"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/butler/manager"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/itchio/ox"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/eos/option"
	"github.com/pkg/errors"
)

func InstallPlan(rc *butlerd.RequestContext, params butlerd.InstallPlanParams) (*butlerd.InstallPlanResult, error) {
	consumer := rc.Consumer
	conn := rc.GetConn()
	defer rc.PutConn(conn)

	game := fetch.LazyFetchGame(rc, params.GameID)
	consumer.Opf("Planning install for %s", operate.GameToString(game))

	runtime := ox.CurrentRuntime()
	baseUploads := fetch.LazyFetchGameUploads(rc, params.GameID)

	// exclude already-installed and currently-installing uploads
	var uploadIDs []interface{}
	for _, u := range baseUploads {
		uploadIDs = append(uploadIDs, u.ID)
	}
	var uploads []*itchio.Upload
	models.MustSelect(conn, &uploads, builder.And(
		builder.In("id", uploadIDs...),
		builder.Expr(`not exists (select 1 from caves where upload_id = uploads.id)`),
		builder.Expr(`not exists (select 1 from downloads where upload_id = uploads.id)`),
	), hades.Search{})
	uploads = manager.NarrowDownUploads(consumer, game, uploads, runtime).Uploads

	res := &butlerd.InstallPlanResult{
		Game:    game,
		Uploads: uploads,
	}

	if len(uploads) == 0 {
		consumer.Statf("No compatible uploads, returning early.")
		return res, nil
	}

	info := &butlerd.InstallPlanInfo{}

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

	operate.LogUpload(consumer, upload, upload.Build)
	info.Upload = upload
	info.Build = upload.Build

	sessionID := params.DownloadSessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
		consumer.Infof("No download session ID passed, using %s", sessionID)
	}

	access := operate.AccessForGameID(conn, game.ID)
	client := rc.Client(access.APIKey)

	installParams := &operate.InstallParams{
		Upload: info.Upload,
		Build:  info.Build,
		Access: access,
	}
	sourceURL := operate.MakeSourceURL(client, consumer, sessionID, installParams, "")

	file, err := eos.Open(sourceURL, option.WithConsumer(consumer))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer file.Close()

	installerInfo, err := installer.GetInstallerInfo(consumer, file)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	info.Type = string(installerInfo.Type)

	// planning is always for a fresh install
	receiptIn := (*bfs.Receipt)(nil)
	installFolder := ""

	dui, err := operate.AssessDiskUsage(file, receiptIn, installFolder, installerInfo)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	info.DiskUsage = &butlerd.DiskUsageInfo{
		FinalDiskUsage:  dui.FinalDiskUsage,
		NeededFreeSpace: dui.NeededFreeSpace,
		Accuracy:        dui.Accuracy.String(),
	}

	res.Info = info
	return res, nil
}
