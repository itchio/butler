package downloads

import (
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/hades"
	"github.com/itchio/butler/database/models"
)

func DownloadsQueue(rc *butlerd.RequestContext, params *butlerd.DownloadsQueueParams) (*butlerd.DownloadsQueueResult, error) {
	consumer := rc.Consumer

	item := params.Item
	if item == nil {
		return nil, errors.Errorf("item cannot be nil")
	}

	startedAt := time.Now().UTC()

	Fresh := false
	_, err := os.Stat(item.InstallFolder)
	if err != nil {
		if os.IsNotExist(err) {
			Fresh = true
		} else {
			return nil, errors.WithStack(err)
		}
	}

	if Fresh {
		consumer.Infof("Downloading over fresh folder")
	} else {
		consumer.Infof("Downloading over existing folder")
	}

	if item.CaveID != "" {
		var downloadsForCaveCount int
		err := rc.DB().Model(&models.Download{}).Where("cave_id = ? AND finished_at IS NULL", item.CaveID).Count(&downloadsForCaveCount).Error
		if err != nil {
			panic(err)
		}

		if downloadsForCaveCount > 0 {
			return nil, errors.Errorf("Already have downloads in progress for %s, refusing to queue another one", operate.GameToString(item.Game))
		}
	}

	// remove other downloads for this cave or this upload
	err = rc.DB().Delete(&models.Download{}, "cave_id = ? OR upload_id = ?", item.CaveID, item.Upload.ID).Error
	if err != nil {
		panic(err)
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

	c := HadesContext(rc)

	err = c.Save(rc.DB(), &hades.SaveParams{
		Record: d,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if item.CaveID != "" {
		// remove other downloads for this cave
		rc.DB().Delete(&models.Download{}, "cave_id = ? and id != ?", item.CaveID, d.ID)
	}

	if params.Item.CaveID != "" && params.Item.Reason == butlerd.DownloadReasonVersionSwitch {
		// if reverting, mark cave as pinned
		cave := models.CaveByID(rc.DB(), params.Item.CaveID)
		cave.Pinned = true
		cave.Save(rc.DB())
	}

	res := &butlerd.DownloadsQueueResult{}
	return res, nil
}
