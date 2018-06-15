package fetch

import (
	"strconv"
	"time"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func FetchProfileGames(rc *butlerd.RequestContext, params *butlerd.FetchProfileGamesParams) (*butlerd.FetchProfileGamesResult, error) {
	consumer := rc.Consumer
	profile, client := rc.ProfileClient(params.ProfileID)

	limit := params.Limit
	if limit == 0 {
		limit = 5
	}
	consumer.Infof("Using limit of %d", limit)
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
		var pgs []*models.ProfileGame
		var cond builder.Cond = builder.Eq{"profile_id": profile.ID}
		var offset int64
		if params.Cursor != "" {
			if parsedOffset, err := strconv.ParseInt(params.Cursor, 10, 64); err == nil {
				offset = parsedOffset
			}
		}
		search := hades.Search().OrderBy("position ASC").Limit(limit + 1).Offset(offset)
		models.MustSelect(conn, &pgs, cond, search)
		models.MustPreload(conn, pgs, hades.Assoc("Game"))

		for i, pg := range pgs {
			if i == len(pgs)-1 && int64(len(pgs)) > limit {
				res.NextCursor = strconv.FormatInt(offset+limit, 10)
			} else {
				res.Items = append(res.Items, FormatProfileGame(pg))
			}
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
