package models

import (
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
	"xorm.io/builder"
)

type UserGameInteraction struct {
	UserID int64 `json:"userId" hades:"primary_key"`
	GameID int64 `json:"gameId" hades:"primary_key"`

	SecondsRun int64      `json:"secondsRun"`
	LastRunAt  *time.Time `json:"lastRunAt"`
	SyncedAt   *time.Time `json:"syncedAt"`
}

func UserGameInteractionByUserAndGame(conn *sqlite.Conn, userID, gameID int64) *UserGameInteraction {
	var ugi UserGameInteraction
	if MustSelectOne(conn, &ugi, builder.Eq{"user_id": userID, "game_id": gameID}) {
		return &ugi
	}
	return nil
}

func SaveUserGameInteractionSummary(conn *sqlite.Conn, userID, gameID int64, summary *itchio.UserGameInteractionsSummary) (retErr error) {
	if summary == nil {
		return nil
	}
	if userID == 0 {
		return errors.New("SaveUserGameInteractionSummary: userID must be non-zero")
	}
	if gameID == 0 {
		return errors.New("SaveUserGameInteractionSummary: gameID must be non-zero")
	}

	defer sqlitex.Save(conn)(&retErr)

	syncedAt := time.Now().UTC()
	if err := Save(conn, &UserGameInteraction{
		UserID:     userID,
		GameID:     gameID,
		SecondsRun: summary.SecondsRun,
		LastRunAt:  summary.LastRunAt,
		SyncedAt:   &syncedAt,
	}); err != nil {
		return err
	}

	// Keep cave fields synchronized for older app versions.
	for _, cave := range CavesByGameID(conn, gameID) {
		cave.UpdateInteractions(summary)
		if err := Save(conn, cave); err != nil {
			return err
		}
	}
	return nil
}
