package downloads

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/hades"
	"xorm.io/builder"
)

func DownloadsClearFinished(rc *butlerd.RequestContext, params butlerd.DownloadsClearFinishedParams) (*butlerd.DownloadsClearFinishedResult, error) {
	// the driver deletes discarded downloads and says so to the client
	defer models.DownloadQueueChanged.Notify()

	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustUpdate(conn, &models.Download{},
			hades.Where(builder.NotNull{"finished_at"}),
			builder.Eq{"discarded": true},
		)
	})

	res := &butlerd.DownloadsClearFinishedResult{}
	return res, nil
}
