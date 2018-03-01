package fetch

import (
	"time"

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

	var games []interface{}
	for _, c := range gamesRes.Games {
		games = append(games, c)
	}

	consumer.Logf("Saving %d games", len(games))

	{
		tx := db.Begin()

		beforeQueue := time.Now()
		err := tx.Model(profile).Association("Games").Replace(games...).Error
		if err != nil {
			tx.Rollback()
			return nil, errors.Wrap(err, 0)
		}
		consumer.Logf("Queuing took %s", time.Since(beforeQueue))

		beforeCommit := time.Now()
		err = tx.Commit().Error
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		consumer.Logf("Commit took %s", time.Since(beforeCommit))
	}

	err = sendDBGames()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.FetchMyGamesResult{}
	return res, nil
}
