package buse

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
	Params OperationParams `json:"params"`
}

// Operation.Resume
type OperationResumeParams struct {
	Params OperationParams `json:"params"`
}

type OperationParams struct {
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
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	ErrorStack   string `json:"errorStack,omitempty"`
}
