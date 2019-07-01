package tasks

import (
	"fmt"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
)

func FetchUserGameSessions(gameID int64) butlerd.BackgroundTask {
	return butlerd.BackgroundTask{
		Desc: fmt.Sprintf("fetch user game sessions for game %d", gameID),
		Do: func(rc *butlerd.RequestContext) error {
			consumer := rc.Consumer
			conn := rc.GetConn()
			defer rc.PutConn(conn)

			caves := models.CavesByGameID(conn, gameID)
			if len(caves) == 0 {
				return nil
			}

			access := operate.AccessForGameID(conn, gameID)
			client := rc.Client(access.APIKey)

			toUpload := models.CaveHistoricalPlayTimeForCaves(conn, caves)
			consumer.Infof("%d historical cave play time pending", len(toUpload))

			for _, playtime := range toUpload {
				consumer.Infof("Syncing historical playtime for cave (%s)", playtime.CaveID)

				_, err := client.CreateUserGameSession(rc.Ctx, itchio.CreateUserGameSessionParams{
					Credentials: access.Credentials,
					GameID:      playtime.GameID,
					UploadID:    playtime.UploadID,
					BuildID:     playtime.BuildID,
					SecondsRun:  playtime.SecondsRun,
				})
				if err != nil {
					consumer.Warnf("Could not sync play time: %+v", err)
				} else {
					playtime.MarkUploaded(conn)
				}
			}

			consumer.Infof("Fetching game interactions summary for game %d...", gameID)
			interactionsRes, err := client.GetGameSessionsSummary(rc.Ctx, gameID)
			if err != nil {
				consumer.Warnf("While fetching user game sessions: %+v", err)
			}

			for _, cave := range caves {
				cave.UpdateInteractions(interactionsRes.Summary)
				cave.Save(conn)
			}

			return nil
		},
	}
}
