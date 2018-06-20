package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
)

func FetchExpireAll(rc *butlerd.RequestContext, params butlerd.FetchExpireAllParams) (*butlerd.FetchExpireAllResult, error) {
	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustDelete(conn, &models.FetchInfo{}, builder.Expr("1"))
	})
	res := &butlerd.FetchExpireAllResult{}
	return res, nil
}
