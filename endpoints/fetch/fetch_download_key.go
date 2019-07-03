package fetch

import (
	"xorm.io/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

func FetchDownloadKey(rc *butlerd.RequestContext, params butlerd.FetchDownloadKeyParams) (*butlerd.FetchDownloadKeyResult, error) {
	conn := rc.GetConn()
	defer rc.PutConn(conn)

	res := &butlerd.FetchDownloadKeyResult{
		Stale: false,
	}

	var dk itchio.DownloadKey
	if models.MustSelectOne(conn, &dk, builder.Eq{"id": params.DownloadKeyID}) {
		res.DownloadKey = &dk
	} else {
		_, err := FetchProfileOwnedKeys(rc, butlerd.FetchProfileOwnedKeysParams{
			ProfileID: params.ProfileID,
			Limit:     1,
			Fresh:     true,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if models.MustSelectOne(conn, &dk, builder.Eq{"id": params.DownloadKeyID}) {
			res.DownloadKey = &dk
		}
	}

	return res, nil
}
