package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/system"
	"github.com/itchio/hades"
	"github.com/itchio/headway/state"
	"xorm.io/builder"
)

func FetchCommons(rc *butlerd.RequestContext, params butlerd.FetchCommonsParams) (*butlerd.FetchCommonsResult, error) {
	conn := rc.GetConn()
	defer rc.PutConn(conn)

	var caves []*butlerd.CaveSummary
	var downloadKeys []*butlerd.DownloadKeySummary
	var installLocations []*models.InstallLocation
	var flocs []*butlerd.InstallLocationSummary

	var interactions map[int64]*butlerd.UserGameInteraction
	if params.ProfileID != 0 {
		if _, err := requireProfile(conn, params.ProfileID); err != nil {
			return nil, err
		}
		interactions = interactionsForUser(conn, params.ProfileID)
	}

	models.MustExec(
		conn,
		builder.Select("id", "game_id", "last_touched_at", "seconds_run", "installed_size",
			"local_seconds_run", "local_last_run_at").From("caves"),
		func(stmt *sqlite.Stmt) error {
			caves = append(caves, &butlerd.CaveSummary{
				ID:              stmt.ColumnText(0),
				GameID:          stmt.ColumnInt64(1),
				LastTouchedAt:   models.ColumnTime(2, stmt),
				SecondsRun:      stmt.ColumnInt64(3),
				InstalledSize:   stmt.ColumnInt64(4),
				LocalSecondsRun: stmt.ColumnInt64(5),
				LocalLastRunAt:  models.ColumnTime(6, stmt),
				Interaction:     interactions[stmt.ColumnInt64(1)],
			})
			return nil
		},
	)

	models.MustExec(
		conn,
		builder.Select("id", "game_id", "created_at").From("download_keys"),
		func(stmt *sqlite.Stmt) error {
			downloadKeys = append(downloadKeys, &butlerd.DownloadKeySummary{
				ID:        stmt.ColumnInt64(0),
				GameID:    stmt.ColumnInt64(1),
				CreatedAt: models.ColumnTime(2, stmt),
			})
			return nil
		},
	)

	models.MustSelect(conn, &installLocations, builder.NewCond(), hades.Search{})
	for _, il := range installLocations {
		flocs = append(flocs, FormatInstallLocation(conn, rc.Consumer, il))
	}

	res := &butlerd.FetchCommonsResult{
		Caves:            caves,
		DownloadKeys:     downloadKeys,
		InstallLocations: flocs,
	}
	return res, nil
}

func FormatInstallLocation(conn *sqlite.Conn, consumer *state.Consumer, il *models.InstallLocation) *butlerd.InstallLocationSummary {
	sum := &butlerd.InstallLocationSummary{
		ID:   il.ID,
		Path: il.Path,
		SizeInfo: &butlerd.InstallLocationSizeInfo{
			InstalledSize: -1,
			FreeSize:      -1,
			TotalSize:     -1,
		},
	}

	models.MustExecRaw(conn, `
		SELECT coalesce(sum(coalesce(installed_size, 0)), 0) AS installed_size
		FROM caves
		WHERE install_location_id = ?
	`, func(stmt *sqlite.Stmt) error {
		sum.SizeInfo.InstalledSize = stmt.ColumnInt64(0)
		return nil
	}, il.ID)

	stats, err := system.StatFS(il.Path)
	if err != nil {
		consumer.Warnf("Could not statFS (%s): %s", il.Path, err.Error())
	} else {
		sum.SizeInfo.FreeSize = stats.FreeSize
		sum.SizeInfo.TotalSize = stats.TotalSize
	}

	return sum
}
