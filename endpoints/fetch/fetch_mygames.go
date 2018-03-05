package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/database/models"
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
		var profileGames []*models.ProfileGame
		err := db.Model(profile).Preload("Game").Related(&profileGames, "ProfileGames").Error
		if err != nil {
			return errors.Wrap(err, 0)
		}

		yn := &buse.FetchMyGamesYieldNotification{
			Offset: 0,
			Total:  int64(len(profileGames)),
			Items:  nil,
		}

		for _, pg := range profileGames {
			yn.Items = append(yn.Items, &buse.MyGame{
				Game:           pg.Game,
				Position:       pg.Position,
				ViewsCount:     pg.ViewsCount,
				DownloadsCount: pg.DownloadsCount,
				PurchasesCount: pg.PurchasesCount,
				Published:      pg.Published,
			})
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

	profile.ProfileGames = nil
	for i, g := range gamesRes.Games {
		profile.ProfileGames = append(profile.ProfileGames, &models.ProfileGame{
			Game:           g,
			Position:       int64(i),
			Published:      g.Published,
			ViewsCount:     g.ViewsCount,
			PurchasesCount: g.PurchasesCount,
			DownloadsCount: g.DownloadsCount,
		})
	}

	err = SaveRecursive(db, consumer, &SaveParams{
		Record: profile,
		Assocs: []string{"ProfileGames"},
	})
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
