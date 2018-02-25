package itchio

// Channel contains information about a channel and its current status
type Channel struct {
	// Name of the channel, usually something like `windows-64-beta` or `osx-universal`
	Name string `json:"name"`
	Tags string `json:"tags"`

	Upload  *Upload `json:"upload"`
	Head    *Build  `json:"head"`
	Pending *Build  `json:"pending"`
}

// A BuildEvent describes something that happened while we were processing a build.
type BuildEvent struct {
	Type    BuildEventType `json:"type"`
	Message string         `json:"message"`
	Data    BuildEventData `json:"data"`
}
