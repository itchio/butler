package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	itchio "github.com/itchio/go-itchio"
)

func Register(router *buse.Router) {
	messages.FetchGame.Register(router, FetchGame)
}

func checkCredentials(credentials *buse.FetchCredentials) error {
	if credentials == nil {
		return errors.New("Credentials must be provided")
	}

	if credentials.SessionID == 0 {
		return errors.New("Credentials.SessionID cannot be zero")
	}

	return nil
}

func FetchGame(rc *buse.RequestContext, params *buse.FetchGameParams) (*buse.FetchGameResult, error) {
	consumer := rc.Consumer

	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if params.GameID == 0 {
		return nil, errors.New("GameID must be non-zero")
	}

	err = checkCredentials(params.Credentials)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	hadLocal := false
	game := &itchio.Game{}
	req := db.Where("id = ?", params.GameID).First(game)
	if req.Error != nil {
		if !req.RecordNotFound() {
			return nil, errors.Wrap(req.Error, 0)
		}
	} else {
		hadLocal = true
		err = messages.FetchGameYield.Notify(rc, &buse.FetchGameYieldNotification{Game: game})
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	client, err := rc.Client(params.Credentials)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	gameRes, err := client.GetGame(&itchio.GetGameParams{
		GameID: params.GameID,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = messages.FetchGameYield.Notify(rc, &buse.FetchGameYieldNotification{Game: gameRes.Game})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if hadLocal {
		consumer.Infof("Updating cached data for game %d", game.ID)
		err = db.Save(gameRes.Game).Error
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	res := &buse.FetchGameResult{}
	return res, nil
}
