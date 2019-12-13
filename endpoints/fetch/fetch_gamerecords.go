package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/hades"
	"xorm.io/builder"
)

func FetchGameRecords(rc *butlerd.RequestContext, params butlerd.FetchGameRecordsParams) (*butlerd.FetchGameRecordsResult, error) {
	consumer := rc.Consumer
	res := &butlerd.FetchGameRecordsResult{}

	rc.WithConn(func(conn *sqlite.Conn) {
		var cond = builder.NewCond()
		var sourceTable string
		search := hades.Search{}

		asc := " ASC"
		desc := " DESC"
		if params.Reverse {
			asc, desc = desc, asc
		}

		titleAZ := func() {
			search = search.OrderBy("lower(games.title)" + asc)
		}

		switch params.Source {
		case butlerd.GameRecordsSourceOwned:
			sourceTable = "download_keys"
			cond = builder.NotNull{"owned"}
			search = search.InnerJoin("games", "games.id = download_keys.game_id")
			switch params.SortBy {
			case "title":
				// A-Z
				titleAZ()
			default: // + "acquiredAt"
				// recent acquisitions first
				search = search.OrderBy("download_keys.created_at" + desc)
			}
		case butlerd.GameRecordsSourceProfile:
			sourceTable = "profile_games"
			search = search.InnerJoin("games", "games.id = profile_games.game_id")
			switch params.SortBy {
			case "views":
				// most viewed first
				search = search.OrderBy("profile_games.views_count" + desc)
			case "downloads":
				// most downloaded first
				search = search.OrderBy("profile_games.downloads_count" + desc)
			case "purchases":
				// most purchased first
				search = search.OrderBy("profile_games.purchases_count" + desc)
			default: // + "title"
				titleAZ()
			}
		case butlerd.GameRecordsSourceCollection:
			sourceTable = "collection_games"
			cond = builder.Eq{"collection_games.collection_id": params.CollectionID}
			search = search.InnerJoin("games", "games.id = collection_games.game_id")
			switch params.SortBy {
			case "title":
				titleAZ()
			default: // + "default"
				// collection's curated order
				search = search.OrderBy("collection_games.position" + asc)
			}
		case butlerd.GameRecordsSourceInstalled:
			sourceTable = "caves"
			search = search.InnerJoin("games", "games.id = caves.game_id")
			switch params.SortBy {
			case "installedSize":
				// biggest first
				search = search.OrderBy("caves.installed_size" + desc)
			case "title":
				titleAZ()
			case "playTime":
				// most played first
				search = search.OrderBy("caves.seconds_run" + desc)
			default: // + "lastTouched"
				search = search.OrderBy("caves.last_touched_at" + desc)
			}
		}

		if params.Filters.Classification != "" {
			cond = builder.And(cond, builder.Eq{"games.classification": params.Filters.Classification})
		}
		if params.Filters.Installed {
			cond = builder.And(cond, builder.NotNull{"installed_at"})
		}
		if params.Filters.Owned {
			cond = builder.And(cond, builder.NotNull{"owned"})
		}

		if sourceTable != "caves" {
			search = search.LeftJoin("caves", "caves.game_id = games.id")
		}
		if sourceTable != "download_keys" {
			search = search.LeftJoin("download_keys", "download_keys.game_id = games.id")
		}

		search = search.GroupBy("games.id")

		limit := params.Limit
		if limit == 0 {
			limit = 5
		}
		search = search.Limit(limit)

		if params.Offset > 0 {
			search = search.Offset(params.Offset)
		}

		// N.B: these need to be kept in the same order as the `GameRecord` struct
		// (the field names don't actuall matter)
		builder := builder.Select(
			"games.id",
			"games.title",
			"coalesce(nullif(games.still_cover_url, ''), games.cover_url) as cover",
			"download_keys.id IS NOT NULL as owned",
			"caves.installed_at AS installed_at",
		).From(sourceTable).Where(cond)

		search.ApplyJoins(builder)

		hcx := models.HadesContext()
		hcx.Consumer = consumer
		hcx.Log = true
		defer func() {
			hcx.Consumer = nil
			hcx.Log = false
		}()
		models.MustExecWithSearch(conn, builder, search, func(stmt *sqlite.Stmt) error {
			return hcx.ScanIntoRows(stmt, &res.Records)
		})
	})

	return res, nil
}
