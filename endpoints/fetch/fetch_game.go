package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/lazyfetch"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
)

func FetchGame(rc *butlerd.RequestContext, params butlerd.FetchGameParams) (*butlerd.FetchGameResult, error) {
	ft := models.FetchTargetForGame(params.GameID)
	res := &butlerd.FetchGameResult{}

	lazyfetch.Do(rc, ft, params, res, func(targets lazyfetch.Targets) {
		var access *operate.GameAccess
		rc.WithConn(func(conn *sqlite.Conn) {
			access = operate.AccessForGameID(conn, params.GameID)
		})
		client := rc.Client(access.APIKey)

		gameRes, err := client.GetGame(itchio.GetGameParams{
			GameID:      params.GameID,
			Credentials: access.Credentials,
		})
		models.Must(err)

		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustSave(conn, gameRes.Game,
				hades.Assoc("Sale"),
				hades.Assoc("User"),
				hades.Assoc("Embed"),
			)
		})
	})

	rc.WithConn(func(conn *sqlite.Conn) {
		res.Game = models.GameByID(conn, params.GameID)
	})

	if res.Game == nil && !params.Fresh {
		params.Fresh = true
		return FetchGame(rc, params)
	}

	return res, nil
}
