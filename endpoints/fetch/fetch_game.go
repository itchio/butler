package fetch

import (
	"time"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func FetchGame(rc *butlerd.RequestContext, params *butlerd.FetchGameParams) (*butlerd.FetchGameResult, error) {
	if params.GameID == 0 {
		return nil, errors.New("gameId must be non-zero")
	}

	consumer := rc.Consumer
	ft := models.FetchTarget{
		Type: "game",
		ID:   params.GameID,
		TTL:  10 * time.Minute,
	}

	fresh := false
	res := &butlerd.FetchGameResult{}

	if params.Fresh {
		consumer.Infof("Doing remote fetch (Fresh specified)")
		fresh = true
	} else if rc.WithConnBool(ft.IsStale) {
		consumer.Infof("Returning stale info")
		res.Stale = true
	}

	if fresh {
		consumer.Debugf("Querying API...")
		var access *operate.GameAccess
		rc.WithConn(func(conn *sqlite.Conn) {
			access = operate.AccessForGameID(conn, params.GameID)
		})
		client := rc.Client(access.APIKey)

		gameRes, err := client.GetGame(itchio.GetGameParams{
			GameID:      params.GameID,
			Credentials: access.Credentials,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustSave(conn, gameRes.Game,
				hades.Assoc("Sale"),
				hades.Assoc("User"),
				hades.Assoc("Embed"),
			)
			// TODO: what about sale/user/embed freshness?
			ft.MarkFresh(conn)
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		res.Game = models.GameByID(conn, params.GameID)
	})

	if res.Game == nil && !params.Fresh {
		freshParams := *params
		freshParams.Fresh = true
		return FetchGame(rc, &freshParams)
	}

	return res, nil
}
