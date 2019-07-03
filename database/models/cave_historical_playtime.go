package models

import (
	"time"

	"crawshaw.io/sqlite"
	"xorm.io/builder"
	"github.com/itchio/hades"
)

type CaveHistoricalPlayTime struct {
	CaveID        string `hades:"primary_key"`
	GameID        int64
	UploadID      int64
	BuildID       int64
	SecondsRun    int64
	LastTouchedAt *time.Time

	CreatedAt  *time.Time
	UploadedAt *time.Time
}

func (chpt *CaveHistoricalPlayTime) MarkUploaded(conn *sqlite.Conn) {
	now := time.Now().UTC()
	chpt.UploadedAt = &now
	MustSave(conn, chpt)
}

func CaveHistoricalPlayTimeForCaves(conn *sqlite.Conn, caves []*Cave) []*CaveHistoricalPlayTime {
	var playtimes []*CaveHistoricalPlayTime
	var caveIDs []interface{}
	for _, cave := range caves {
		caveIDs = append(caveIDs, cave.ID)
	}
	MustSelect(conn, &playtimes, builder.And(
		builder.IsNull{"uploaded_at"},
		builder.In("cave_id", caveIDs...),
	), hades.Search{})
	return playtimes
}
