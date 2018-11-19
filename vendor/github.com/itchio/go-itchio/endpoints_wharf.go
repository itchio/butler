package itchio

import (
	"encoding/json"

	"github.com/pkg/errors"
)

//-------------------------------------------------------

// WharfStatusResponse is what the API responds with when we ask for
// the status of the wharf infrastructure
type WharfStatusResponse struct {
	Success bool `json:"success"`
}

// WharfStatus requests the status of the wharf infrastructure
func (c *Client) WharfStatus() (*WharfStatusResponse, error) {
	q := NewQuery(c, "/wharf/status")
	r := &WharfStatusResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

// ListChannelsResponse is what the API responds with when we ask for all the
// channels of a particular game
type ListChannelsResponse struct {
	Channels map[string]*Channel `json:"channels"`
}

// Channel contains information about a channel and its current status
type Channel struct {
	// Name of the channel, usually something like `windows-64-beta` or `osx-universal`
	Name string `json:"name"`
	Tags string `json:"tags"`

	Upload  *Upload `json:"upload"`
	Head    *Build  `json:"head"`
	Pending *Build  `json:"pending"`
}

// ListChannels returns a list of the channels for a game
func (c *Client) ListChannels(target string) (*ListChannelsResponse, error) {
	q := NewQuery(c, "/wharf/channels")
	q.AddString("target", target)
	r := &ListChannelsResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

// GetChannelResponse is what the API responds with when we ask info about a channel
type GetChannelResponse struct {
	Channel *Channel `json:"channel"`
}

// GetChannel returns information about a given channel for a given game
func (c *Client) GetChannel(target string, channel string) (*GetChannelResponse, error) {
	q := NewQuery(c, "/wharf/channels/%s", channel)
	q.AddString("target", target)
	r := &GetChannelResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

// CreateBuildParams : params for CreateBuild
type CreateBuildParams struct {
	Target      string
	Channel     string
	UserVersion string
}

// CreateBuildResponse : response for CreateBuild
type CreateBuildResponse struct {
	Build struct {
		ID          int64 `json:"id"`
		UploadID    int64 `json:"uploadId"`
		ParentBuild struct {
			ID int64 `json:"id"`
		} `json:"parentBuild"`
	}
}

// CreateBuild creates a new build for a given user/game:channel, with
// an optional user version
func (c *Client) CreateBuild(p CreateBuildParams) (*CreateBuildResponse, error) {
	q := NewQuery(c, "/wharf/builds")
	q.AddString("target", p.Target)
	q.AddString("channel", p.Channel)
	q.AddStringIfNonEmpty("user_version", p.UserVersion)
	r := &CreateBuildResponse{}
	return r, q.Post(r)
}

//-------------------------------------------------------

// ListBuildFilesResponse : response for ListBuildFiles
type ListBuildFilesResponse struct {
	Files []*BuildFile `json:"files"`
}

// ListBuildFiles returns a list of files associated to a build
func (c *Client) ListBuildFiles(buildID int64) (*ListBuildFilesResponse, error) {
	q := NewQuery(c, "/wharf/builds/%d/files", buildID)
	r := &ListBuildFilesResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

// FileUploadType describes which strategy is used for uploading to storage
// some types allow for uploading in blocks (which is resumable), some
// expect the whole payload in one request.
type FileUploadType string

const (
	// FileUploadTypeMultipart lets you send metadata + all the content in a single request
	FileUploadTypeMultipart FileUploadType = "multipart"
	// FileUploadTypeResumable lets you send blocks of N*128KB at a time. The upload session is
	// started from the API server, so the ingest point will be anchored wherever the API server is.
	FileUploadTypeResumable FileUploadType = "resumable"
	// FileUploadTypeDeferredResumable also lets you send blocks of N*128KB at a time, but it
	// lets you start the upload session from the client, which means you might get a closer ingest point.
	FileUploadTypeDeferredResumable FileUploadType = "deferred_resumable"
)

// FileUploadSpec contains the info needed to upload one specific build file
type FileUploadSpec struct {
	ID            int64             `json:"id"`
	UploadURL     string            `json:"uploadUrl"`
	UploadParams  map[string]string `json:"uploadParams"`
	UploadHeaders map[string]string `json:"uploadHeaders"`
}

// CreateBuildFileParams : params for CreateBuildFile
type CreateBuildFileParams struct {
	BuildID        int64
	Type           BuildFileType
	SubType        BuildFileSubType
	FileUploadType FileUploadType
	Filename       string
}

// CreateBuildFileResponse : response for CreateBuildFile
type CreateBuildFileResponse struct {
	File *FileUploadSpec `json:"file"`
}

// CreateBuildFile creates a new build file for a build.
func (c *Client) CreateBuildFile(p CreateBuildFileParams) (*CreateBuildFileResponse, error) {
	q := NewQuery(c, "/wharf/builds/%d/files", p.BuildID)
	q.AddString("type", string(p.Type))
	q.AddStringIfNonEmpty("sub_type", string(p.SubType))
	q.AddStringIfNonEmpty("upload_type", string(p.FileUploadType))
	q.AddStringIfNonEmpty("filename", p.Filename)
	r := &CreateBuildFileResponse{}
	return r, q.Post(r)
}

//-------------------------------------------------------

// FinalizeBuildFileParams : params for FinalizeBuildFile
type FinalizeBuildFileParams struct {
	BuildID int64
	FileID  int64
	Size    int64
}

// FinalizeBuildFileResponse : response for FinalizeBuildFile
type FinalizeBuildFileResponse struct{}

// FinalizeBuildFile marks the end of the upload for a build file.
// It validates that the size of the file in storage is the same
// we pass to this API call.
func (c *Client) FinalizeBuildFile(p FinalizeBuildFileParams) (*FinalizeBuildFileResponse, error) {
	q := NewQuery(c, "/wharf/builds/%d/files/%d", p.BuildID, p.FileID)
	q.AddInt64("size", p.Size)
	r := &FinalizeBuildFileResponse{}
	return r, q.Post(r)
}

//-------------------------------------------------------

var (
	// ErrBuildFileNotFound is returned when someone is asking for a non-existent file
	ErrBuildFileNotFound = errors.New("build file not found in storage")
)

// MakeBuildFileDownloadURLParams : params for MakeBuildFileDownloadURL
type MakeBuildFileDownloadURLParams struct {
	BuildID int64
	FileID  int64
}

// MakeBuildFileDownloadURL returns a download URL for a given build file
func (c *Client) MakeBuildFileDownloadURL(p MakeBuildFileDownloadURLParams) string {
	q := NewQuery(c, "/wharf/builds/%d/files/%d/download", p.BuildID, p.FileID)
	q.AddAPICredentials()
	return q.URL()
}

//-------------------------------------------------------

// CreateBuildEventParams : params for CreateBuildEvent
type CreateBuildEventParams struct {
	BuildID int64
	Type    BuildEventType
	Message string
	Data    BuildEventData
}

// BuildEventType specifies what kind of event a build event is - could be a log message, etc.
type BuildEventType string

const (
	// BuildEventLog is for build events of type log message
	BuildEventLog BuildEventType = "log"
)

// BuildEventData is a JSON object associated with a build event
type BuildEventData map[string]interface{}

// CreateBuildEventResponse is what the API responds with when you create a new build event
type CreateBuildEventResponse struct{}

// CreateBuildEvent associates a new build event to a build
func (c *Client) CreateBuildEvent(p CreateBuildEventParams) (*CreateBuildEventResponse, error) {
	q := NewQuery(c, "/wharf/builds/%d/events", p.BuildID)
	q.AddString("type", string(p.Type))
	q.AddString("message", p.Message)

	jsonData, err := json.Marshal(p.Data)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	q.AddString("data", string(jsonData))
	r := &CreateBuildEventResponse{}
	return r, q.Post(r)
}

//-------------------------------------------------------

// CreateBuildFailureParams : params for CreateBuildFailure
type CreateBuildFailureParams struct {
	BuildID int64
	Message string
	Fatal   bool
}

// CreateBuildFailureResponse : response for CreateBuildFailure
type CreateBuildFailureResponse struct{}

// CreateBuildFailure marks a given build as failed. We get to specify an error message and
// if it's a fatal error (if not, the build can be retried after a bit)
func (c *Client) CreateBuildFailure(p CreateBuildFailureParams) (*CreateBuildFailureResponse, error) {
	q := NewQuery(c, "/wharf/builds/%d/failures", p.BuildID)
	q.AddString("message", p.Message)
	q.AddBoolIfTrue("fatal", p.Fatal)
	r := &CreateBuildFailureResponse{}
	return r, q.Post(r)
}

//-------------------------------------------------------

// CreateRediffBuildFailureParams : params for CreateRediffBuildFailure
type CreateRediffBuildFailureParams struct {
	BuildID int64
	Message string
}

// CreateRediffBuildFailureResponse : response for CreateRediffBuildFailure
type CreateRediffBuildFailureResponse struct{}

// CreateRediffBuildFailure marks a given build as having failed to rediff (optimize)
func (c *Client) CreateRediffBuildFailure(p CreateRediffBuildFailureParams) (*CreateRediffBuildFailureResponse, error) {
	q := NewQuery(c, "/wharf/builds/%d/failures/rediff", p.BuildID)
	q.AddString("message", p.Message)
	r := &CreateRediffBuildFailureResponse{}
	return r, q.Post(r)
}

//-------------------------------------------------------

// ListBuildEventsResponse is what the API responds with when we ask for the list of events for a build
type ListBuildEventsResponse struct {
	Events []*BuildEvent `json:"events"`
}

// A BuildEvent describes something that happened while we were processing a build.
type BuildEvent struct {
	Type    BuildEventType `json:"type"`
	Message string         `json:"message"`
	Data    BuildEventData `json:"data"`
}

// ListBuildEvents returns a series of events associated with a given build
func (c *Client) ListBuildEvents(buildID int64) (*ListBuildEventsResponse, error) {
	q := NewQuery(c, "/wharf/builds/%d/events", buildID)
	r := &ListBuildEventsResponse{}
	return r, q.Get(r)
}
