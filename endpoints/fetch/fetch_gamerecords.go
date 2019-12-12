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
		var cond builder.Cond = builder.And(builder.NotNull{"owned"}, builder.NotNull{"installed_at"})
		search := hades.Search{}
		search = search.LeftJoin("caves", "caves.game_id = games.id").LeftJoin("download_keys", "download_keys.game_id = games.id").GroupBy("games.id").Limit(5)

		builder := builder.Select(
			"games.id",
			"games.title",
			"coalesce(nullif(games.still_cover_url, ''), games.cover_url) as cover",
			"download_keys.id IS NOT NULL as owned",
			"caves.installed_at AS installed_at",
		).From("games").Where(cond)

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
