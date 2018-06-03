package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func FetchGame(rc *butlerd.RequestContext, params *butlerd.FetchGameParams) (*butlerd.FetchGameResult, error) {
	consumer := rc.Consumer

	if params.GameID == 0 {
		return nil, errors.New("gameId must be non-zero")
	}

	sendDBGame := func() error {
		var game *itchio.Game
		rc.WithConn(func(conn *sqlite.Conn) {
			game = models.GameByID(conn, params.GameID)
		})
		if game != nil {
			err := messages.FetchGameYield.Notify(rc, &butlerd.FetchGameYieldNotification{Game: game})
			if err != nil {
				return errors.WithStack(err)
			}
		}
		return nil
	}

	err := sendDBGame()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	consumer.Debugf("Querying API...")
	var access *operate.GameAccess
	rc.WithConn(func(conn *sqlite.Conn) {
		access = operate.AccessForGameID(conn, params.GameID)
	})
	client := rc.Client(access.APIKey)

	gameRes, err := client.GetGame(&itchio.GetGameParams{
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
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = sendDBGame()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := &butlerd.FetchGameResult{}
	return res, nil
}
