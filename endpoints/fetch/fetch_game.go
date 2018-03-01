package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	itchio "github.com/itchio/go-itchio"
)

func FetchGame(rc *buse.RequestContext, params *buse.FetchGameParams) (*buse.FetchGameResult, error) {
	consumer := rc.Consumer

	if params.GameID == 0 {
		return nil, errors.New("gameId must be non-zero")
	}

	client, err := rc.SessionClient(params.SessionID)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	sendDBGame := func() error {
		game := &itchio.Game{}
		req := db.Where("id = ?", params.GameID).First(game)
		if req.Error != nil {
			if !req.RecordNotFound() {
				return errors.Wrap(req.Error, 0)
			}
		} else {
			err = messages.FetchGameYield.Notify(rc, &buse.FetchGameYieldNotification{Game: game})
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
		return nil
	}

	err = sendDBGame()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Infof("Querying API...")
	gameRes, err := client.GetGame(&itchio.GetGameParams{
		GameID: params.GameID,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = db.Save(gameRes.Game).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = sendDBGame()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.FetchGameResult{}
	return res, nil
}
