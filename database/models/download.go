package models

import (
	"time"

	itchio "github.com/itchio/go-itchio"
)

type Download struct {
	// An UUID
	ID string `json:"id"`

	Reason     string    `json:"reason"`
	Progress   float64   `json:"progress"`
	Finished   bool      `json:"finished"`
	Order      int64     `json:"order"`
	BPS        float64   `json:"bps"`
	ETA        float64   `json:"eta"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	Err        string    `json:"err"`
	ErrStack   string    `json:"errStack"`

	CaveID          string `json:"caveId"`
	Game            JSON   `json:"game"`
	Upload          JSON   `json:"upload"`
	Build           JSON   `json:"build"`
	TotalSize       int64  `json:"totalSize"`
	InstallLocation string `json:"installLocation"`
	InstallFolder   string `json:"installFolder"`
}

func (d *Download) SetGame(game *itchio.Game) error { return MarshalGame(game, &d.Game) }
func (d *Download) GetGame() (*itchio.Game, error)  { return UnmarshalGame(d.Game) }

func (d *Download) SetUpload(upload *itchio.Upload) error { return MarshalUpload(upload, &d.Upload) }
func (d *Download) GetUpload() (*itchio.Upload, error)    { return UnmarshalUpload(d.Upload) }

func (d *Download) SetBuild(build *itchio.Build) error { return MarshalBuild(build, &d.Build) }
func (d *Download) GetBuild() (*itchio.Build, error)   { return UnmarshalBuild(d.Build) }
