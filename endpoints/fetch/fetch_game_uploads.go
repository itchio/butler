package fetch

import (
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/lazyfetch"
	"github.com/itchio/butler/manager"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/itchio/ox"
)

func FetchGameUploads(rc *butlerd.RequestContext, params butlerd.FetchGameUploadsParams) (*butlerd.FetchGameUploadsResult, error) {
	ft := models.FetchTargetForGameUploads(params.GameID)
	res := &butlerd.FetchGameUploadsResult{}
	conn := rc.GetConn()
	defer rc.PutConn(conn)

	lazyfetch.Do(rc, ft, params, res, func(targets lazyfetch.Targets) {
		access := operate.AccessForGameID(conn, params.GameID)
		client := rc.Client(access.APIKey)

		uploadsRes, err := client.ListGameUploads(rc.Ctx, itchio.ListGameUploadsParams{
			GameID:      params.GameID,
			Credentials: access.Credentials,
		})
		models.Must(err)

		var validUploadIDs []interface{}
		var gameUploads []*models.GameUpload
		for i, u := range uploadsRes.Uploads {
			targets.Add(models.FetchTargetForUpload(u.ID))
			validUploadIDs = append(validUploadIDs, u.ID)
			gameUploads = append(gameUploads, &models.GameUpload{
				GameID:   params.GameID,
				UploadID: u.ID,
				Upload:   u,
				Position: int64(i),
			})
		}

		// TODO: do that in transaction?
		models.MustDelete(conn, &models.GameUpload{}, builder.And(
			builder.Eq{"game_id": params.GameID},
			builder.NotIn("upload_id", validUploadIDs...),
		))
		models.MustSave(conn, gameUploads,
			hades.Assoc("Upload",
				hades.Assoc("Build"),
			),
		)
	})

	var gameUploads []*models.GameUpload
	models.MustSelect(conn, &gameUploads, builder.Eq{
		"game_id": params.GameID,
	}, hades.Search{}.OrderBy("position ASC"))
	models.MustPreload(conn, gameUploads,
		hades.Assoc("Upload",
			hades.Assoc("Build"),
		),
	)

	var uploads []*itchio.Upload
	for _, gu := range gameUploads {
		uploads = append(uploads, gu.Upload)
	}

	if params.OnlyCompatible {
		game := LazyFetchGame(rc, params.GameID)
		runtime := ox.CurrentRuntime()
		narrowRes := manager.NarrowDownUploads(rc.Consumer, game, uploads, runtime)
		uploads = narrowRes.Uploads
	}

	res.Uploads = uploads

	return res, nil
}

func LazyFetchGameUploads(rc *butlerd.RequestContext, gameID int64) []*itchio.Upload {
	var uploadsRes *butlerd.FetchGameUploadsResult
	err := lazyfetch.EnsureFresh(&uploadsRes, func(fresh bool) (lazyfetch.LazyFetchResponse, error) {
		return FetchGameUploads(rc, butlerd.FetchGameUploadsParams{
			GameID: gameID,
			Fresh:  fresh,
		})
	})
	models.Must(err)
	return uploadsRes.Uploads
}
