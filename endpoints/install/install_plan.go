package install

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/endpoints/fetch"
	"github.com/itchio/butler/manager"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/ox"
)

func InstallPlan(rc *butlerd.RequestContext, params butlerd.InstallPlanParams) (*butlerd.InstallPlanResult, error) {
	consumer := rc.Consumer

	game := fetch.LazyFetchGame(rc, params.GameID)
	consumer.Opf("Planning install for %s", operate.GameToString(game))

	runtime := ox.CurrentRuntime()
	uploads := fetch.LazyFetchGameUploads(rc, params.GameID)
	uploads = manager.NarrowDownUploads(consumer, game, uploads, runtime).Uploads

	res := &butlerd.InstallPlanResult{
		Game:    game,
		Uploads: uploads,
	}

	if len(uploads) == 0 {
		consumer.Statf("No compatible uploads, returning early.")
		return res, nil
	}

	var upload *itchio.Upload
	if params.UploadID == 0 {
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

	consumer.Infof("Stub: not returning install info")

	return res, nil
}
