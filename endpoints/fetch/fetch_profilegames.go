package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/database/hades"
	"github.com/itchio/butler/database/models"
)

func FetchProfileGames(rc *buse.RequestContext, params *buse.FetchProfileGamesParams) (*buse.FetchProfileGamesResult, error) {
	consumer := rc.Consumer

	profile, client, err := rc.ProfileClient(params.ProfileID)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	c := hades.NewContext(db, consumer)

	sendDBGames := func() error {
		var profileGames []*models.ProfileGame
		err := db.Model(profile).Preload("Game").Related(&profileGames, "ProfileGames").Error
		if err != nil {
			return errors.Wrap(err, 0)
		}

		yn := &buse.FetchProfileGamesYieldNotification{
			Offset: 0,
			Total:  int64(len(profileGames)),
			Items:  nil,
		}

		for _, pg := range profileGames {
			yn.Items = append(yn.Items, &buse.ProfileGame{
				Game:           pg.Game,
				Position:       pg.Position,
				ViewsCount:     pg.ViewsCount,
				DownloadsCount: pg.DownloadsCount,
				PurchasesCount: pg.PurchasesCount,
				Published:      pg.Published,
			})
		}

		err = messages.FetchProfileGamesYield.Notify(rc, yn)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}

	err = sendDBGames()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Debugf("Querying API...")

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

	err = c.Save(db, &hades.SaveParams{
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

	res := &buse.FetchProfileGamesResult{}
	return res, nil
}
