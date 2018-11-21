package fetch

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/lazyfetch"
	"github.com/itchio/butler/endpoints/tasks"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
)

func FetchGame(rc *butlerd.RequestContext, params butlerd.FetchGameParams) (*butlerd.FetchGameResult, error) {
	ft := models.FetchTargetForGame(params.GameID)
	res := &butlerd.FetchGameResult{}
	conn := rc.GetConn()
	defer rc.PutConn(conn)

	lazyfetch.Do(rc, ft, params, res, func(targets lazyfetch.Targets) {
		rc.QueueBackgroundTask(tasks.FetchUserGameSessions(params.GameID))

		access := operate.AccessForGameID(conn, params.GameID)
		client := rc.Client(access.APIKey)

		gameRes, err := client.GetGame(itchio.GetGameParams{
			GameID:      params.GameID,
			Credentials: access.Credentials,
		})
		models.Must(err)

		models.MustSave(conn, gameRes.Game,
			hades.Assoc("Sale"),
			hades.Assoc("User"),
			hades.Assoc("Embed"),
		)
	})

	res.Game = models.GameByID(conn, params.GameID)

	if res.Game == nil && !params.Fresh {
		params.Fresh = true
		return FetchGame(rc, params)
	}

	models.MustPreloadGameSales(conn, res.Game)
	return res, nil
}

func LazyFetchGame(rc *butlerd.RequestContext, gameID int64) *itchio.Game {
	var gameRes *butlerd.FetchGameResult
	err := lazyfetch.EnsureFresh(&gameRes, func(fresh bool) (lazyfetch.LazyFetchResponse, error) {
		return FetchGame(rc, butlerd.FetchGameParams{
			GameID: gameID,
			Fresh:  fresh,
		})
	})
	models.Must(err)
	return gameRes.Game
}
