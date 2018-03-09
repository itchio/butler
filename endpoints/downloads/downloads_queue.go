package downloads

import (
	"time"

	"github.com/go-errors/errors"

	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/hades"
	"github.com/itchio/butler/database/models"
)

func DownloadsQueue(rc *buse.RequestContext, params *buse.DownloadsQueueParams) (*buse.DownloadsQueueResult, error) {
	item := params.Item
	if item == nil {
		return nil, errors.Errorf("item cannot be nil")
	}

	startedAt := time.Now().UTC()
	d := &models.Download{
		ID:            item.ID,
		CaveID:        item.CaveID,
		Position:      models.DownloadMaxPosition(rc.DB()) + 1,
		Game:          item.Game,
		Upload:        item.Upload,
		Build:         item.Build,
		StagingFolder: item.StagingFolder,
		StartedAt:     &startedAt,
	}

	err := HadesContext(rc).Save(rc.DB(), &hades.SaveParams{
		Record: d,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.DownloadsQueueResult{}
	return res, nil
}
