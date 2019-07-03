package fetch

import (
	"time"

	"crawshaw.io/sqlite"
	"xorm.io/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
)

func FetchSale(rc *butlerd.RequestContext, params butlerd.FetchSaleParams) (*butlerd.FetchSaleResult, error) {
	res := &butlerd.FetchSaleResult{}

	rc.WithConn(func(conn *sqlite.Conn) {
		var sales []*itchio.Sale
		now := time.Now().UTC().Format(time.RFC3339Nano)
		cond := builder.And(
			builder.Eq{"game_id": params.GameID},
			builder.Gt{"end_date": now},
		)
		search := hades.Search{}.OrderBy("rate DESC")
		models.MustSelect(conn, &sales, cond, search)

		if len(sales) > 0 {
			res.Sale = sales[0]
		}
	})
	return res, nil
}
