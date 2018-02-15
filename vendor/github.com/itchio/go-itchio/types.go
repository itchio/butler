package itchio

import (
	"fmt"
	"strings"

	"github.com/go-errors/errors"
)

// User represents an itch.io account, with basic profile info
type User struct {
	ID            int64  `json:"id"`
	Username      string `json:"username"`
	CoverURL      string `json:"coverUrl"`
	StillCoverURL string `json:"stillCoverUrl"`
}

// Game represents a page on itch.io, it could be a game,
// a tool, a comic, etc.
type Game struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`

	Title     string `json:"title"`
	ShortText string `json:"shortText"`
	Type      string `json:"type"`

	CoverURL      string `json:"coverUrl"`
	StillCoverURL string `json:"stillCoverUrl"`

	CreatedAt   string `json:"createdAt"`
	PublishedAt string `json:"publishedAt"`

	MinPrice      int64 `json:"minPrice"`
	InPressSystem bool  `json:"inPressSystem"`
	HasDemo       bool  `json:"hasDemo"`

	Classification string `json:"classification"`

	OSX     bool `json:"pOsx"`
	Linux   bool `json:"pLinux"`
	Windows bool `json:"pWindows"`
	Android bool `json:"pAndroid"`
}

// An Upload is a downloadable file
type Upload struct {
	ID          int64  `json:"id"`
	Filename    string `json:"filename"`
	DisplayName string `json:"displayName"`
	Size        int64  `json:"size"`
	ChannelName string `json:"channelName"`
	Build       *Build `json:"build"`
	Demo        bool   `json:"demo"`
	Preorder    bool   `json:"preorder"`

	Type    string `json:"type"`
	OSX     bool   `json:"pOsx"`
	Linux   bool   `json:"pLinux"`
	Windows bool   `json:"pWindows"`
	Android bool   `json:"pAndroid"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// BuildFile contains information about a build's "file", which could be its
// archive, its signature, its patch, etc.
type BuildFile struct {
	ID      int64            `json:"id"`
	Size    int64            `json:"size"`
	State   BuildFileState   `json:"state"`
	Type    BuildFileType    `json:"type"`
	SubType BuildFileSubType `json:"subType"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// Build contains information about a specific build
type Build struct {
	ID            int64      `json:"id"`
	ParentBuildID int64      `json:"parentBuildId"`
	State         BuildState `json:"state"`

	Version     int64  `json:"version"`
	UserVersion string `json:"userVersion"`

	Files []*BuildFile `json:"files"`

	User      User   `json:"user"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// Channel contains information about a channel and its current status
type Channel struct {
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

type APIError struct {
	Messages []string
}

var _ error = (*APIError)(nil)

func (ae *APIError) Error() string {
	return fmt.Sprintf("itch.io API error: %s", strings.Join(ae.Messages, ", "))
}

// IsApiError returns true if an error is an itch.io API error,
// even if it's wrapped with github.com/go-errors/errors
func IsAPIError(err error) bool {
	if err == nil {
		return false
	}

	if se, ok := err.(*errors.Error); ok {
		return IsAPIError(se.Err)
	}

	_, ok := err.(*APIError)
	return ok
}
