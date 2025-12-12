package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"xorm.io/builder"
)

func FetchExpireAll(rc *butlerd.RequestContext, params butlerd.FetchExpireAllParams) (*butlerd.FetchExpireAllResult, error) {
	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustDelete(conn, &models.FetchInfo{}, builder.Expr("1"))
	})
	res := &butlerd.FetchExpireAllResult{}
	return res, nil
}
