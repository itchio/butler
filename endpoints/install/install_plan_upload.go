package install

import (
	itchio "github.com/itchio/go-itchio"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"

	"github.com/itchio/hades"

	"github.com/pkg/errors"
	"xorm.io/builder"
)

func InstallPlanUpload(rc *butlerd.RequestContext, params butlerd.InstallPlanUploadParams) (*butlerd.InstallPlanUploadResult, error) {
	if params.ID != "" {
		_, cleanup := rc.MakeCancelable(params.ID)
		defer cleanup()
	}

	conn := rc.GetConn()
	defer rc.PutConn(conn)

	var upload itchio.Upload
	if ok := models.MustSelectOne(conn, &upload, builder.Eq{"id": params.UploadID}); !ok {
		return nil, errors.Errorf("upload %d not found", params.UploadID)
	}
	models.MustPreload(conn, &upload, hades.Assoc("Build"))

	var gameUpload models.GameUpload
	if ok := models.MustSelectOne(conn, &gameUpload, builder.Eq{"upload_id": params.UploadID}); !ok {
		return nil, errors.Errorf("game upload mapping not found for upload %d", params.UploadID)
	}

	info, err := getPlanInfo(rc, conn, &upload, gameUpload.GameID, params.DownloadSessionID)
	if err != nil {
		return nil, err
	}

	return &butlerd.InstallPlanUploadResult{
		Info: info,
	}, nil
}
