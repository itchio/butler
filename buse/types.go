package buse

import (
	itchio "github.com/itchio/go-itchio"
)

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
// Game
//----------------------------------------------------------------------

// Game.FindUploads
type GameFindUploadsParams struct {
	Game        *itchio.Game     `json:"game"`
	Credentials *GameCredentials `json:"credentials"`
}

type GameFindUploadsResult struct {
	Uploads []*itchio.Upload `json:"uploads"`
}

//----------------------------------------------------------------------
// Operation
//----------------------------------------------------------------------

type Operation string

var (
	OperationInstall   Operation = "install"
	OperationUninstall Operation = "uninstall"
)

// Operation.Start
type OperationStartParams struct {
	ID            string    `json:"id"`
	StagingFolder string    `json:"stagingFolder"`
	Operation     Operation `json:"operation"`

	// this is more or less a union, the relevant field
	// should be set depending on the 'Operation' type
	InstallParams   *InstallParams   `json:"installParams,omitempty"`
	UninstallParams *UninstallParams `json:"uninstallParams,omitempty"`
}

// Operation.Cancel
type OperationCancelParams struct {
	ID string `json:"id"`
}

type OperationCancelResult struct{}

// InstallParams contains all the parameters needed to perform
// an installation for a game
type InstallParams struct {
	Game          *itchio.Game     `json:"game"`
	InstallFolder string           `json:"installFolder"`
	Upload        *itchio.Upload   `json:"upload"`
	Build         *itchio.Build    `json:"build"`
	Credentials   *GameCredentials `json:"credentials"`
	Fresh         bool             `json:"fresh"`
}

type UninstallParams struct {
	InstallFolder string `json:"installFolder"`
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

// Operation.Progress
// Sent periodically to inform on the current state an operation
type OperationProgressNotification struct {
	Progress float64 `json:"progress"`
	ETA      float64 `json:"eta"`
	BPS      float64 `json:"bps"`
}

type TaskReason string

const (
	TaskReasonInstall   TaskReason = "install"
	TaskReasonReinstall TaskReason = "reinstall"
	TaskReasonUpdate    TaskReason = "update"
	TaskReasonRevert    TaskReason = "revert"
	TaskReasonHeal      TaskReason = "heal"
	TaskReasonUninstall TaskReason = "uninstall"
)

type TaskType string

const (
	TaskTypeDownload  TaskType = "download"
	TaskTypeInstall   TaskType = "install"
	TaskTypeUninstall TaskType = "uninstall"
)

type TaskStartedNotification struct {
	Reason    TaskReason     `json:"reason"`
	Type      TaskType       `json:"type"`
	Game      *itchio.Game   `json:"game"`
	Upload    *itchio.Upload `json:"upload"`
	Build     *itchio.Build  `json:"build,omitempty"`
	TotalSize int64          `json:"totalSize,omitempty"`
}

type TaskEndedNotification struct {
}

// Result for
//   - Operation.Start
type OperationResult struct {
	ID            string      `json:"id"`
	Success       bool        `json:"success"`
	InstallResult interface{} `json:"installResult,omitempty"`
	ErrorMessage  string      `json:"errorMessage,omitempty"`
	ErrorStack    string      `json:"errorStack,omitempty"`
}

type InstallResult struct {
	Game   *itchio.Game   `json:"game"`
	Upload *itchio.Upload `json:"upload"`
	Build  *itchio.Build  `json:"build"`
	// TODO: verdict ?
}

// Log
type LogNotification struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// Test.DoubleTwice
type TestDoubleTwiceRequest struct {
	Number int64 `json:"number"`
}

// Result for Test.DoubleTwice
type TestDoubleTwiceResult struct {
	Number int64 `json:"number"`
}

// Test.Double
type TestDoubleRequest struct {
	Number int64 `json:"number"`
}

// Result for Test.Double
type TestDoubleResult struct {
	Number int64 `json:"number"`
}

const (
	CodeOperationCancelled = 499
)
