package models

import (
	"time"

	"github.com/go-errors/errors"
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

	InstalledAt   time.Time `json:"installedAt"`
	LastTouchedAt time.Time `json:"lastTouchedAt"`
	SecondsRun    int64     `json:"secondsRun"`

	Verdict       JSON  `json:"verdict"`
	InstalledSize int64 `json:"installedSize"`

	InstallLocation string     `json:"installLocation"`
	InstallFolder   string     `json:"installFolder"`
	PathScheme      PathScheme `json:"pathScheme"`
}

type PathScheme int64

const (
	PathSchemeLegacyPerUser PathScheme = 1
	ModernShared            PathScheme = 2
)

func (c *Cave) SetVerdict(verdict *configurator.Verdict) error {
	return MarshalVerdict(verdict, &c.Verdict)
}
func (c *Cave) GetVerdict() (*configurator.Verdict, error) { return UnmarshalVerdict(c.Verdict) }

func CaveByID(db *gorm.DB, id string) (*Cave, error) {
	c := &Cave{}
	req := db.Where("id = ?", id).First(c)
	if req.Error != nil {
		if req.RecordNotFound() {
			return nil, nil
		}
		return nil, errors.Wrap(req.Error, 0)
	}

	return c, nil
}

func CavesByGameID(db *gorm.DB, gameID int64) ([]*Cave, error) {
	var cs []*Cave
	err := db.Where("game_id = ?", gameID).Find(&cs).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	return cs, nil
}

func (c *Cave) Touch() {
	c.LastTouchedAt = time.Now().UTC()
}

func (c *Cave) RecordPlayTime(playTime time.Duration) {
	c.SecondsRun += int64(playTime.Seconds())
	c.Touch()
}
