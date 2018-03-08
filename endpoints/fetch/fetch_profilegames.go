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

	profile, client := rc.ProfileClient(params.ProfileID)

	c := HadesContext(rc)

	sendDBGames := func() error {
		c.Preload(rc.DB(), &hades.PreloadParams{
			Record: profile,
			Fields: []hades.PreloadField{
				{Name: "ProfileGames", OrderBy: `"position" ASC`},
				{Name: "ProfileGames.Game"},
			},
		})
		profileGames := profile.ProfileGames

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

		err := messages.FetchProfileGamesYield.Notify(rc, yn)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}

	err := sendDBGames()
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

	err = c.Save(rc.DB(), &hades.SaveParams{
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
