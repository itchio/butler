package install

import (
	"context"

	itchio "github.com/itchio/go-itchio"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch"
	"github.com/itchio/butler/manager"

	"github.com/itchio/hades"

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

	conn := rc.GetConn()
	defer rc.PutConn(conn)

	game := fetch.LazyFetchGame(rc, params.GameID)
	if err := checkCancelled(rc.Ctx); err != nil {
		return nil, err
	}
	consumer.Opf("Planning install for %s", operate.GameToString(game))

	baseUploads := fetch.LazyFetchGameUploads(rc, params.GameID)
	if err := checkCancelled(rc.Ctx); err != nil {
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

	return res, nil
}
