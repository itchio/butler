package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/pager"
	"github.com/itchio/hades"
	"xorm.io/builder"
)

func FetchCaves(rc *butlerd.RequestContext, params butlerd.FetchCavesParams) (*butlerd.FetchCavesResult, error) {
	res := &butlerd.FetchCavesResult{}
	var err error
	rc.WithConn(func(conn *sqlite.Conn) {
		if params.ProfileID != 0 {
			if _, err = requireProfile(conn, params.ProfileID); err != nil {
				return
			}
		}
		fetchCavesWithConn(conn, params, res)
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func fetchCavesWithConn(conn *sqlite.Conn, params butlerd.FetchCavesParams, res *butlerd.FetchCavesResult) {
	var cond = builder.NewCond()
	joinGames := false
	joinInteractions := false
	search := hades.Search{}

	switch params.SortBy {
	case "title":
		ordering := pager.Ordering("ASC", params.Reverse)
		search = search.OrderBy("lower(games.title) " + ordering)
		joinGames = true
	case "playTime":
		ordering := pager.Ordering("DESC", params.Reverse)
		if params.ProfileID != 0 {
			search = search.OrderBy("coalesce(user_game_interactions.seconds_run, 0) " + ordering)
			joinInteractions = true
		} else {
			search = search.OrderBy("caves.seconds_run " + ordering)
		}
	case "installedAt":
		ordering := pager.Ordering("DESC", params.Reverse)
		search = search.OrderBy("caves.installed_at " + ordering)
	case "installedSize":
		ordering := pager.Ordering("DESC", params.Reverse)
		search = search.OrderBy("caves.installed_size " + ordering)
	case "lastTouched", "":
		ordering := pager.Ordering("DESC", params.Reverse)
		if params.ProfileID != 0 {
			search = search.OrderBy("coalesce(user_game_interactions.last_run_at, caves.installed_at) " + ordering)
			joinInteractions = true
		} else {
			search = search.OrderBy("coalesce(caves.last_touched_at, caves.installed_at) " + ordering)
		}
	}

	if params.Filters.Classification != "" {
		cond = builder.And(cond, builder.Eq{"games.classification": params.Filters.Classification})
		joinGames = true
	}

	if params.Filters.InstallLocationID != "" {
		cond = builder.And(cond, builder.Eq{"caves.install_location_id": params.Filters.InstallLocationID})
	}

	if params.Filters.GameID != 0 {
		cond = builder.And(cond, builder.Eq{"caves.game_id": params.Filters.GameID})
	}

	if params.Filters.NeverPlayed {
		if params.ProfileID != 0 {
			joinInteractions = true
			cond = builder.And(cond,
				builder.IsNull{"user_game_interactions.last_run_at"},
				builder.Expr("coalesce(user_game_interactions.seconds_run, 0) = 0"),
			)
		} else {
			// check seconds_run too: a legacy scan import can leave seconds_run
			// > 0 with a null last_touched_at, which isn't "never played".
			cond = builder.And(cond,
				builder.IsNull{"caves.last_touched_at"},
				builder.Eq{"caves.seconds_run": 0},
				builder.Expr("coalesce(caves.local_seconds_run, 0) = 0"),
			)
		}
	}

	if params.Search != "" {
		cond = builder.And(cond, builder.Like{"games.title", params.Search})
		joinGames = true
	}

	if joinGames {
		search = search.InnerJoin("games", "games.id = caves.game_id")
	}
	if joinInteractions {
		search = search.LeftJoin(interactionsJoin(params.ProfileID))
	}

	var items []*models.Cave
	pg := pager.New(params)
	res.NextCursor = pg.Fetch(conn, &items, cond, search)
	models.PreloadCaves(conn, items)

	var interactions map[int64]*butlerd.UserGameInteraction
	if params.ProfileID != 0 {
		interactions = interactionsForUser(conn, params.ProfileID)
	}
	for _, cave := range items {
		formatted := FormatCave(conn, cave)
		formatted.Interaction = interactions[cave.GameID]
		res.Items = append(res.Items, formatted)
	}
}
