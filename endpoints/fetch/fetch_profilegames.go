package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/lazyfetch"
	"github.com/itchio/butler/endpoints/fetch/pager"
	"github.com/itchio/hades"
)

func FetchProfileGames(rc *butlerd.RequestContext, params butlerd.FetchProfileGamesParams) (*butlerd.FetchProfileGamesResult, error) {
	profile, client := rc.ProfileClient(params.ProfileID)

	ft := models.FetchTargetForProfileGames(profile.ID)
	res := &butlerd.FetchProfileGamesResult{}

	lazyfetch.Do(rc, ft, params, res, func(targets lazyfetch.Targets) {
		gamesRes, err := client.ListProfileGames(rc.Ctx)
		models.Must(err)

		profile.ProfileGames = nil
		for i, g := range gamesRes.Games {
			targets.Add(models.FetchTargetForGame(g.ID))
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
		})
	})

	rc.WithConn(func(conn *sqlite.Conn) {
		var cond builder.Cond = builder.Eq{"profile_games.profile_id": profile.ID}
		joinGames := false
		search := hades.Search{}

		switch params.SortBy {
		case "title":
			joinGames = true
			search = search.OrderBy("games.title " + pager.Ordering("ASC", params.Reverse))
		case "views":
			search = search.OrderBy("profile_games.views_count " + pager.Ordering("DESC", params.Reverse))
		case "downloads":
			search = search.OrderBy("profile_games.downloads_count " + pager.Ordering("DESC", params.Reverse))
		case "purchases":
			search = search.OrderBy("profile_games.purchases_count " + pager.Ordering("DESC", params.Reverse))
		case "default", "":
			search = search.OrderBy("profile_games.position " + pager.Ordering("ASC", params.Reverse))
		}

		switch params.Filters.Visibility {
		case "draft":
			cond = builder.And(cond, builder.Eq{"profile_games.published": 0})
		case "published":
			cond = builder.And(cond, builder.Eq{"profile_games.published": 1})
		}

		switch params.Filters.PaidStatus {
		case "free":
			joinGames = true
			cond = builder.And(cond, builder.Eq{"games.min_price": 0})
		case "paid":
			joinGames = true
			cond = builder.And(cond, builder.Neq{"games.min_price": 0})
		}

		if params.Search != "" {
			joinGames = true
			cond = builder.And(cond, builder.Like{"games.title", params.Search})
		}

		if joinGames {
			search = search.Join("games", "games.id = profile_games.game_id")
		}

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
