package fetch

import (
	"time"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/pager"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func FetchProfileGames(rc *butlerd.RequestContext, params *butlerd.FetchProfileGamesParams) (*butlerd.FetchProfileGamesResult, error) {
	consumer := rc.Consumer
	profile, client := rc.ProfileClient(params.ProfileID)

	ft := models.FetchTarget{
		Type: "profile_games",
		ID:   profile.ID,
		TTL:  10 * time.Minute,
	}

	fresh := false
	res := &butlerd.FetchProfileGamesResult{}

	if params.Fresh {
		consumer.Infof("Doing remote fetch (Fresh specified)")
		fresh = true
	} else if rc.WithConnBool(ft.MustIsStale) {
		consumer.Infof("Returning stale results")
		res.Stale = true
	}

	if fresh {
		fts := []models.FetchTarget{ft}

		consumer.Debugf("Querying API...")

		gamesRes, err := client.ListProfileGames()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		profile.ProfileGames = nil
		for i, g := range gamesRes.Games {
			fts = append(fts, models.FetchTarget{
				Type: "game",
				ID:   g.ID,
				TTL:  10 * time.Minute,
			})
			profile.ProfileGames = append(profile.ProfileGames, &models.ProfileGame{
				Game:           g,
				Position:       int64(i),
				Published:      g.Published,
				ViewsCount:     g.ViewsCount,
				PurchasesCount: g.PurchasesCount,
				DownloadsCount: g.DownloadsCount,
			})
		}

		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustSave(conn, profile,
				hades.OmitRoot(),
				hades.AssocReplace("ProfileGames",
					hades.Assoc("Game"),
				),
			)
			models.MustMarkAllFresh(conn, fts)
		})
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		var cond builder.Cond = builder.Eq{"profile_id": profile.ID}
		search := hades.Search{}.OrderBy("position ASC")

		var items []*models.ProfileGame
		pg := pager.New(params)
		res.NextCursor = pg.Fetch(conn, &items, cond, search)
		models.MustPreload(conn, items, hades.Assoc("Game"))
		for _, item := range items {
			res.Items = append(res.Items, FormatProfileGame(item))
		}
	})
	return res, nil
}

func FormatProfileGame(pg *models.ProfileGame) *butlerd.ProfileGame {
	return &butlerd.ProfileGame{
		Game: pg.Game,

		ViewsCount:     pg.ViewsCount,
		DownloadsCount: pg.DownloadsCount,
		PurchasesCount: pg.PurchasesCount,

		Published: pg.Published,
	}
}
