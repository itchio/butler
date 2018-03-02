package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
)

func FetchMyGames(rc *buse.RequestContext, params *buse.FetchMyGamesParams) (*buse.FetchMyGamesResult, error) {
	consumer := rc.Consumer

	client, err := rc.SessionClient(params.SessionID)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	profile := &models.Profile{}
	err = db.Where("id = ?", params.SessionID).First(profile).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	sendDBGames := func() error {
		var games []*itchio.Game
		err := db.Model(profile).Related(&games, "Games").Error
		if err != nil {
			return errors.Wrap(err, 0)
		}

		yn := &buse.FetchMyGamesYieldNotification{
			Offset: 0,
			Total:  int64(len(games)),
			Items:  games,
		}
		err = messages.FetchMyGamesYield.Notify(rc, yn)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}

	err = sendDBGames()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Infof("Querying API...")

	gamesRes, err := client.ListMyGames()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = SaveRecursive(db, consumer, gamesRes.Games, nil)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = sendDBGames()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.FetchMyGamesResult{}
	return res, nil
}
