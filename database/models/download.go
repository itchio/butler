package models

import (
	"fmt"
	"time"

	"xorm.io/builder"
	"github.com/itchio/hades"

	"crawshaw.io/sqlite"
	itchio "github.com/itchio/go-itchio"
)

type Download struct {
	// An UUID
	ID string `json:"id" hades:"primary_key"`

	Reason     string     `json:"reason"`
	Position   int64      `json:"position"`
	StartedAt  *time.Time `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`

	// Full error, complete with stack trace etc.
	Error *string `json:"error"`
	// Standard butlerd error code
	ErrorCode *int64 `json:"errorCode"`
	// Short error message (hopefully human-readable)
	ErrorMessage *string `json:"errorMessage"`

	CaveID string `json:"caveId"`

	GameID int64        `json:"gameId"`
	Game   *itchio.Game `json:"game"`

	UploadID int64          `json:"uploadId"`
	Upload   *itchio.Upload `json:"upload"`

	BuildID int64         `json:"buildId"`
	Build   *itchio.Build `json:"build"`

	StagingFolder string `json:"stagingFolder"`
	InstallFolder string `json:"installFolder"`

	InstallLocationID string `json:"installLocationId"`

	Discarded bool `json:"discarded"`
	Fresh     bool `json:"fresh"`
}

func AllDownloads(conn *sqlite.Conn) []*Download {
	var dls []*Download
	MustSelect(conn, &dls, builder.Not{builder.Expr("discarded")}, hades.Search{}.OrderBy("position ASC"))
	return dls
}

func DownloadByID(conn *sqlite.Conn, downloadID string) *Download {
	var dl Download
	if MustSelectOne(conn, &dl, builder.Eq{"id": downloadID}) {
		return &dl
	}
	return nil
}

func (d *Download) Preload(conn *sqlite.Conn) {
	if d != nil {
		PreloadDownloads(conn, d)
	}
}

func PreloadDownloads(conn *sqlite.Conn, downloadOrDownloads interface{}) {
	MustPreload(conn, downloadOrDownloads,
		hades.Assoc("Game"),
		hades.Assoc("Upload"),
		hades.Assoc("Build"),
	)
}

func DownloadMinPosition(conn *sqlite.Conn) int64 {
	return downloadExtremePosition(conn, "min")
}

func DownloadMaxPosition(conn *sqlite.Conn) int64 {
	return downloadExtremePosition(conn, "max")
}

func downloadExtremePosition(conn *sqlite.Conn, extreme string) int64 {
	var position int64

	q := fmt.Sprintf(`SELECT coalesce(%s(position), 0) AS position FROM downloads`, extreme)
	MustExecRaw(conn, q, func(stmt *sqlite.Stmt) error {
		position = stmt.ColumnInt64(0)
		return nil
	})
	return position
}

func (d *Download) Save(conn *sqlite.Conn) {
	MustSave(conn, d)
}

func DiscardDownloadsByCaveID(conn *sqlite.Conn, caveID string) {
	MustUpdate(conn, &Download{},
		hades.Where(builder.Eq{"cave_id": caveID}),
		builder.Eq{"discarded": true},
	)
}
