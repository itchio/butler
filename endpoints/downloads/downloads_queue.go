package downloads

import (
	"os"
	"time"

	"github.com/go-errors/errors"

	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/hades"
	"github.com/itchio/butler/database/models"
)

func DownloadsQueue(rc *buse.RequestContext, params *buse.DownloadsQueueParams) (*buse.DownloadsQueueResult, error) {
	consumer := rc.Consumer

	item := params.Item
	if item == nil {
		return nil, errors.Errorf("item cannot be nil")
	}

	startedAt := time.Now().UTC()

	Fresh := true
	_, err := os.Stat(item.InstallFolder)
	if err != nil {
		if os.IsNotExist(err) {
			Fresh = false
		} else {
			return nil, errors.Wrap(err, 0)
		}
	}

	if Fresh {
		consumer.Infof("Downloading over fresh folder")
	} else {
		consumer.Infof("Downloading over existing folder")
	}

	d := &models.Download{
		ID:            item.ID,
		Reason:        string(item.Reason),
		CaveID:        item.CaveID,
		Position:      models.DownloadMaxPosition(rc.DB()) + 1,
		Game:          item.Game,
		Upload:        item.Upload,
		Build:         item.Build,
		InstallFolder: item.InstallFolder,
		StagingFolder: item.StagingFolder,
		StartedAt:     &startedAt,
		Fresh:         Fresh,
	}

	err = HadesContext(rc).Save(rc.DB(), &hades.SaveParams{
		Record: d,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.DownloadsQueueResult{}
	return res, nil
}
