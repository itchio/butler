package models

import (
	"time"

	"crawshaw.io/sqlite"
	"github.com/itchio/dash"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"xorm.io/builder"
)

type Cave struct {
	ID string `json:"id"`

	GameID int64        `json:"gameId"`
	Game   *itchio.Game `json:"game"`

	ExternalGameID int64 `json:"externalGameId"`

	UploadID int64          `json:"uploadId"`
	Upload   *itchio.Upload `json:"upload"`
	BuildID  int64          `json:"buildId"`
	Build    *itchio.Build  `json:"build"`

	Morphing bool `json:"morphing"`
	Pinned   bool `json:"pinned"`

	InstalledAt   *time.Time `json:"installedAt"`
	LastTouchedAt *time.Time `json:"lastTouchedAt"`
	SecondsRun    int64      `json:"secondsRun"`

	SnoozedAt *time.Time `json:"snoozedAt"`

	Verdict       JSON  `json:"verdict"`
	InstalledSize int64 `json:"installedSize"`

	InstallLocationID string           `json:"installLocationId"`
	InstallLocation   *InstallLocation `json:"installLocation"`

	InstallFolderName string `json:"installFolderName"`

	// If set, InstallLocationID is empty and this is used
	// for all operations instead
	CustomInstallFolder string `json:"customInstallFolder"`
}

func (c *Cave) SetVerdict(verdict *dash.Verdict) {
	err := MarshalVerdict(verdict, &c.Verdict)
	if err != nil {
		panic(err)
	}
}
func (c *Cave) GetVerdict() *dash.Verdict {
	v, err := UnmarshalVerdict(c.Verdict)
	if err != nil {
		panic(err)
	}
	return v
}

func CaveByID(conn *sqlite.Conn, id string) *Cave {
	var c Cave
	if MustSelectOne(conn, &c, builder.Eq{"id": id}) {
		return &c
	}
	return nil
}

func CavesByGameID(conn *sqlite.Conn, gameID int64) []*Cave {
	var cs []*Cave
	MustSelect(conn, &cs, builder.Eq{"game_id": gameID}, hades.Search{})
	return cs
}

func (c *Cave) Touch() {
	lastTouchedAt := time.Now().UTC()
	c.LastTouchedAt = &lastTouchedAt
}

func (c *Cave) UpdateInstallTime() {
	installedAt := time.Now().UTC()
	c.InstalledAt = &installedAt
}

func (c *Cave) RecordPlayTime(playTime time.Duration) {
	c.SecondsRun += int64(playTime.Seconds())
	c.Touch()
}

func (c *Cave) UpdateInteractions(summary *itchio.UserGameInteractionsSummary) {
	c.SecondsRun = summary.SecondsRun
	if summary.LastRunAt != nil {
		c.LastTouchedAt = summary.LastRunAt
	}
}

func (c *Cave) GetInstallLocation(conn *sqlite.Conn) *InstallLocation {
	if c.InstallLocation != nil {
		return c.InstallLocation
	}

	MustPreload(conn, c, hades.Assoc("InstallLocation"))
	return c.InstallLocation
}

func (c *Cave) GetInstallFolder(conn *sqlite.Conn) string {
	if c.CustomInstallFolder != "" {
		return c.CustomInstallFolder
	}

	return c.GetInstallLocation(conn).GetInstallFolder(c.InstallFolderName)
}

func (c *Cave) Preload(conn *sqlite.Conn) {
	if c == nil {
		return
	}
	PreloadCaves(conn, c)
}

func PreloadCaves(conn *sqlite.Conn, caveOrCaves interface{}) {
	MustPreload(conn, caveOrCaves,
		hades.Assoc("Game"),
		hades.Assoc("Upload"),
		hades.Assoc("Build"),
		hades.Assoc("InstallLocation"),
	)
}

func (c *Cave) Save(conn *sqlite.Conn) {
	MustSave(conn, c)
}

func (c *Cave) SaveWithAssocs(conn *sqlite.Conn) {
	MustSave(conn, c,
		hades.Assoc("Game"),
		hades.Assoc("Upload"),
		hades.Assoc("Build"),
	)
}

func (c *Cave) Delete(conn *sqlite.Conn) {
	MustDelete(conn, &Cave{}, builder.Eq{"id": c.ID})
}
