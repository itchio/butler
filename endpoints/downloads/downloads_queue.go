package downloads

import (
	"os"
	"time"

	"github.com/go-xorm/builder"
	"github.com/pkg/errors"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/hades"
)

func DownloadsQueue(rc *butlerd.RequestContext, params butlerd.DownloadsQueueParams) (*butlerd.DownloadsQueueResult, error) {
	consumer := rc.Consumer
	conn := rc.GetConn()
	defer rc.PutConn(conn)

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
		downloadsForCaveCount := models.MustCount(conn, &models.Download{},
			builder.And(
				builder.Eq{"cave_id": item.CaveID},
				builder.IsNull{"finished_at"},
			),
		)

		if downloadsForCaveCount > 0 {
			return nil, errors.Errorf("Already have downloads in progress for %s, refusing to queue another one", operate.GameToString(item.Game))
		}
	}

	// remove other downloads for this cave or this upload
	models.MustDelete(conn, &models.Download{},
		builder.Or(
			builder.Eq{"cave_id": item.CaveID},
			builder.Eq{"upload_id": item.Upload.ID},
		),
	)

	d := &models.Download{
		ID:                item.ID,
		Reason:            string(item.Reason),
		CaveID:            item.CaveID,
		Position:          models.DownloadMaxPosition(conn) + 1,
		Game:              item.Game,
		Upload:            item.Upload,
		Build:             item.Build,
		InstallFolder:     item.InstallFolder,
		StagingFolder:     item.StagingFolder,
		InstallLocationID: item.InstallLocationID,
		StartedAt:         &startedAt,
		Fresh:             Fresh,
	}

	models.MustSave(conn, d,
		hades.Assoc("Game"),
		hades.Assoc("Upload"),
		hades.Assoc("Build"),
	)

	if item.CaveID != "" {
		// remove other downloads for this cave
		models.MustDelete(conn, &models.Download{},
			builder.And(
				builder.Eq{"cave_id": item.CaveID},
				builder.Neq{"id": d.ID},
			),
		)
	}

	if params.Item.CaveID != "" && params.Item.Reason == butlerd.DownloadReasonVersionSwitch {
		// if reverting, mark cave as pinned
		cave := models.CaveByID(conn, params.Item.CaveID)
		cave.Pinned = true
		cave.Save(conn)
	}

	res := &butlerd.DownloadsQueueResult{}
	return res, nil
}
