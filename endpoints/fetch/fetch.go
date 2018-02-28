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

func FetchGame(rc *buse.RequestContext, params *buse.FetchGameParams) (*buse.FetchGameResult, error) {
	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	game := &itchio.Game{}
	err = db.Where("id = ?", params.GameID).First(game).Error
	if err != nil {
		if !db.RecordNotFound() {
			return nil, errors.Wrap(err, 0)
		}
	} else {
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

	res := &buse.FetchGameResult{}
	return res, nil
}
