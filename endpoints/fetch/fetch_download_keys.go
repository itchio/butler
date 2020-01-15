package fetch

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
	"xorm.io/builder"
)

func FetchDownloadKeys(rc *butlerd.RequestContext, params butlerd.FetchDownloadKeysParams) (*butlerd.FetchDownloadKeysResult, error) {
	conn := rc.GetConn()
	defer rc.PutConn(conn)

	ft := models.FetchTargetForProfileOwnedKeys(params.ProfileID)
	res := &butlerd.FetchDownloadKeysResult{
		Stale: ft.MustIsStale(conn),
	}

	if params.Fresh {
		_, err := FetchProfileOwnedKeys(rc, butlerd.FetchProfileOwnedKeysParams{
			ProfileID: params.ProfileID,
			Limit:     1,
			Fresh:     true,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}
		res.Stale = false
	}

	var cond = builder.NewCond()
	if params.Filters.GameID != 0 {
		cond = builder.Eq{"game_id": params.Filters.GameID}
	}

	var dks []*itchio.DownloadKey

	var search = hades.Search{}
	limit := params.Limit
	if limit == 0 {
		limit = 5
	}
	search = search.Limit(limit)

	if params.Offset > 0 {
		search = search.Offset(params.Offset)
	}

	models.MustSelect(conn, &dks, cond, search)
	res.Items = dks

	return res, nil
}
