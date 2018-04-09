package fetch

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/hades"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

func FetchGame(rc *butlerd.RequestContext, params *butlerd.FetchGameParams) (*butlerd.FetchGameResult, error) {
	consumer := rc.Consumer

	if params.GameID == 0 {
		return nil, errors.New("gameId must be non-zero")
	}

	sendDBGame := func() error {
		game := models.GameByID(rc.DB(), params.GameID)
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
	creds := operate.CredentialsForGameID(rc.DB(), params.GameID)

	client := rc.ClientFromCredentials(creds)

	gameRes, err := client.GetGame(&itchio.GetGameParams{
		GameID: params.GameID,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	c := HadesContext(rc)

	err = c.Save(rc.DB(), &hades.SaveParams{
		Record: gameRes.Game,
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
