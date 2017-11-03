package buse

import itchio "github.com/itchio/go-itchio"

// must be kept in sync with clients, see for example
// https://github.com/itchio/node-butler

//----------------------------------------------------------------------
// Version
//----------------------------------------------------------------------

// Version.Get
type VersionGetParams struct{}

// Result for Version.Get
type VersionGetResult struct {
	// Something short, like `v8.0.0`
	Version string `json:"version"`

	// Something long, like `v8.0.0, built on Aug 27 2017 @ 01:13:55, ref d833cc0aeea81c236c81dffb27bc18b2b8d8b290`
	VersionString string `json:"versionString"`
}

//----------------------------------------------------------------------
// Operation
//----------------------------------------------------------------------

// Operation.Start
type OperationStartParams struct {
	StagingFolder string         `json:"stagingFolder"`
	Operation     string         `json:"operation"`
	InstallParams *InstallParams `json:"installParams,omitempty"`
}

// InstallParams contains all the parameters needed to perform
// an installation for a game
type InstallParams struct {
	Game          *itchio.Game     `json:"game"`
	InstallFolder string           `json:"installFolder"`
	Upload        *itchio.Upload   `json:"upload"`
	Build         *itchio.Build    `json:"build"`
	Credentials   *GameCredentials `json:"credentials"`
}

// GameCredentials contains all the credentials required to make API requests
// including the download key if any
type GameCredentials struct {
	Server      string `json:"server"`
	APIKey      string `json:"apiKey"`
	DownloadKey int64  `json:"downloadKey"`
}

type PickUploadParams struct {
	Uploads []*itchio.Upload `json:"uploads"`
}

type PickUploadResult struct {
	Index int64 `json:"index"`
}

// Operation.Resume
type OperationResumeParams struct {
	StagingFolder string `json:"stagingFolder"`
}

// Operation.Progress
// Sent periodically to inform on the current state an operation
type OperationProgressNotification struct {
	Progress float64 `json:"progress"`
	ETA      float64 `json:"eta,omitempty"`
	BPS      int64   `json:"bps,omitempty"`
}

// Result for
//   - Operation.Start
//   - Operation.Resume
type OperationResult struct {
	Success       bool        `json:"success"`
	InstallResult interface{} `json:"installResult,omitempty"`
	ErrorMessage  string      `json:"errorMessage,omitempty"`
	ErrorStack    string      `json:"errorStack,omitempty"`
}

// Log
type LogNotification struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}
