package fetch

import (
	"fmt"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
	"xorm.io/builder"
)

func FetchGameInteraction(rc *butlerd.RequestContext, params butlerd.FetchGameInteractionParams) (*butlerd.FetchGameInteractionResult, error) {
	var apiKey string
	var err error
	rc.WithConn(func(conn *sqlite.Conn) {
		var profile *models.Profile
		profile, err = requireProfile(conn, params.ProfileID)
		if profile != nil {
			apiKey = profile.APIKey
		}
	})
	if err != nil {
		return nil, err
	}

	if params.Fresh {
		client := rc.Client(apiKey)
		summaryRes, err := client.GetGameSessionsSummary(rc.Ctx, params.GameID)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		rc.WithConn(func(conn *sqlite.Conn) {
			err = models.SaveUserGameInteractionSummary(conn, params.ProfileID, params.GameID, summaryRes.Summary)
		})
		if err != nil {
			return nil, err
		}
	}

	res := &butlerd.FetchGameInteractionResult{}
	rc.WithConn(func(conn *sqlite.Conn) {
		row := models.UserGameInteractionByUserAndGame(conn, params.ProfileID, params.GameID)
		res.Interaction = formatInteraction(row)
		res.Stale = row == nil
	})
	return res, nil
}

func requireProfile(conn *sqlite.Conn, profileID int64) (*models.Profile, error) {
	profile := models.ProfileByID(conn, profileID)
	if profile == nil {
		return nil, errors.WithStack(butlerd.CodeNoSuchProfile)
	}
	return profile, nil
}

func formatInteraction(row *models.UserGameInteraction) *butlerd.UserGameInteraction {
	if row == nil {
		return nil
	}
	return &butlerd.UserGameInteraction{
		UserID:     row.UserID,
		GameID:     row.GameID,
		SecondsRun: row.SecondsRun,
		LastRunAt:  row.LastRunAt,
		SyncedAt:   row.SyncedAt,
	}
}

// The join condition is keyed against caves.game_id, so it only composes
// with queries that select from or join the caves table.
func interactionsJoin(profileID int64) (table string, cond string) {
	return "user_game_interactions", fmt.Sprintf(
		"user_game_interactions.game_id = caves.game_id AND user_game_interactions.user_id = %d",
		profileID)
}

func interactionsForUser(conn *sqlite.Conn, userID int64) map[int64]*butlerd.UserGameInteraction {
	var rows []*models.UserGameInteraction
	models.MustSelect(conn, &rows, builder.Eq{"user_id": userID}, hades.Search{})
	byGame := make(map[int64]*butlerd.UserGameInteraction, len(rows))
	for _, row := range rows {
		byGame[row.GameID] = formatInteraction(row)
	}
	return byGame
}
