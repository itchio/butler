package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/lazyfetch"
	itchio "github.com/itchio/go-itchio"
)

func FetchUser(rc *butlerd.RequestContext, params butlerd.FetchUserParams) (*butlerd.FetchUserResult, error) {
	ft := models.FetchTargetForUser(params.UserID)
	res := &butlerd.FetchUserResult{}

	lazyfetch.Do(rc, ft, params, res, func(targets lazyfetch.Targets) {
		_, client := rc.ProfileClient(params.ProfileID)

		userRes, err := client.GetUser(rc.Ctx, itchio.GetUserParams{
			UserID: params.UserID,
		})
		models.Must(err)

		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustSave(conn, userRes.User)
		})
	})

	rc.WithConn(func(conn *sqlite.Conn) {
		res.User = models.UserByID(conn, params.UserID)
	})

	if res.User == nil && !params.Fresh {
		params.Fresh = true
		return FetchUser(rc, params)
	}

	return res, nil
}
