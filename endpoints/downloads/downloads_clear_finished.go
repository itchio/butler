package downloads

import (
	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/hades"
)

func DownloadsClearFinished(rc *butlerd.RequestContext, params *butlerd.DownloadsClearFinishedParams) (*butlerd.DownloadsClearFinishedResult, error) {
	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustUpdate(conn, &models.Download{},
			hades.Where(builder.NotNull{"finished_at"}),
			builder.Eq{"discarded": true},
		)
	})

	res := &butlerd.DownloadsClearFinishedResult{}
	return res, nil
}
