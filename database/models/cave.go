package models

import (
	"time"

	"github.com/itchio/butler/configurator"
	itchio "github.com/itchio/go-itchio"
	"github.com/jinzhu/gorm"
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

	InstalledAt   *time.Time `json:"installedAt"`
	LastTouchedAt *time.Time `json:"lastTouchedAt"`
	SecondsRun    int64      `json:"secondsRun"`

	Verdict       JSON  `json:"verdict"`
	InstalledSize int64 `json:"installedSize"`

	InstallLocationID string           `json:"installLocationId"`
	InstallLocation   *InstallLocation `json:"installLocation"`

	InstallFolderName string `json:"installFolderName"`

	// If set, InstallLocationID is empty and this is used
	// for all operations instead
	CustomInstallFolder string `json:"customInstallFolder"`
}

func (c *Cave) SetVerdict(verdict *configurator.Verdict) {
	err := MarshalVerdict(verdict, &c.Verdict)
	if err != nil {
		panic(err)
	}
}
func (c *Cave) GetVerdict() *configurator.Verdict {
	v, err := UnmarshalVerdict(c.Verdict)
	if err != nil {
		panic(err)
	}
	return v
}

func CaveByID(db *gorm.DB, id string) *Cave {
	var c Cave
	req := db.Where("id = ?", id).First(&c)
	if req.Error != nil {
		if req.RecordNotFound() {
			return nil
		}
		panic(req.Error)
	}

	return &c
}

func CavesByGameID(db *gorm.DB, gameID int64) []*Cave {
	var cs []*Cave
	err := db.Where("game_id = ?", gameID).Find(&cs).Error
	if err != nil {
		panic(err)
	}
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

func (c *Cave) GetInstallLocation(db *gorm.DB) *InstallLocation {
	if c.InstallLocation != nil {
		return c.InstallLocation
	}

	MustPreloadSimple(db, c, "InstallLocation")
	return c.InstallLocation
}

func (c *Cave) GetInstallFolder(db *gorm.DB) string {
	if c.CustomInstallFolder != "" {
		return c.CustomInstallFolder
	}

	return c.GetInstallLocation(db).GetInstallFolder(c.InstallFolderName)
}

func (c *Cave) Preload(db *gorm.DB) {
	if c == nil {
		return
	}
	PreloadCaves(db, c)
}

func PreloadCaves(db *gorm.DB, caveOrCaves interface{}) {
	MustPreloadSimple(db, caveOrCaves, "Game", "Upload", "Build", "InstallLocation")
}

func (c *Cave) Save(db *gorm.DB) {
	err := db.Save(c).Error
	if err != nil {
		panic(err)
	}
}
