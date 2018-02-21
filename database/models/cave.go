package models

import (
	"time"

	"github.com/itchio/butler/configurator"
	itchio "github.com/itchio/go-itchio"
)

type Cave struct {
	ID string `json:"id"`

	GameID         int64 `json:"gameId"`
	ExternalGameID int64 `json:"externalGameId"`

	Upload   JSON `json:"upload"`
	Build    JSON `json:"build"`
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

func (c *Cave) SetUpload(upload *itchio.Upload) error { return MarshalUpload(upload, &c.Upload) }
func (c *Cave) GetUpload() (*itchio.Upload, error)    { return UnmarshalUpload(c.Upload) }

func (c *Cave) SetBuild(build *itchio.Build) error { return MarshalBuild(build, &c.Build) }
func (c *Cave) GetBuild() (*itchio.Build, error)   { return UnmarshalBuild(c.Build) }

func (c *Cave) SetVerdict(verdict *configurator.Verdict) error {
	return MarshalVerdict(verdict, &c.Verdict)
}
func (c *Cave) GetVerdict() (*configurator.Verdict, error) { return UnmarshalVerdict(c.Verdict) }
