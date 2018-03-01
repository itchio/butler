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

	CaveID          string         `json:"caveId"`
	GameID          int64          `json:"gameId"`
	Game            *itchio.Game   `json:"game"`
	UploadID        int64          `json:"uploadId"`
	Upload          *itchio.Upload `json:"upload"`
	BuildID         int64          `json:"buildId"`
	Build           *itchio.Build  `json:"build"`
	TotalSize       int64          `json:"totalSize"`
	InstallLocation string         `json:"installLocation"`
	InstallFolder   string         `json:"installFolder"`
}
