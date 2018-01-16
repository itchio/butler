package itchio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-errors/errors"
)

// A Client allows consuming the itch.io API
type Client struct {
	Key           string
	HTTPClient    *http.Client
	BaseURL       string
	RetryPatterns []time.Duration
	UserAgent     string
}

func defaultRetryPatterns() []time.Duration {
	return []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
	}
}

// ClientWithKey creates a new itch.io API client with a given API key
func ClientWithKey(key string) *Client {
	c := &Client{
		Key:           key,
		HTTPClient:    http.DefaultClient,
		RetryPatterns: defaultRetryPatterns(),
		UserAgent:     "go-itchio",
	}
	c.SetServer("https://itch.io")
	return c
}

// SetServer allows changing the server to which we're making API
// requests (which defaults to the reference itch.io server)
func (c *Client) SetServer(itchioServer string) *Client {
	c.BaseURL = fmt.Sprintf("%s/api/1", itchioServer)
	return c
}

// WharfStatus requests the status of the wharf infrastructure
func (c *Client) WharfStatus() (r WharfStatusResponse, err error) {
	path := c.MakePath("wharf/status")

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

// GetMe returns information about to which the current credentials belong
func (c *Client) GetMe() (GetMeResponse, error) {
	path := c.MakePath("me")

	var r GetMeResponse

	resp, err := c.Get(path)
	if err != nil {
		return r, errors.Wrap(err, 0)
	}

	err = ParseAPIResponse(&r, resp)
	if err != nil {
		return r, errors.Wrap(err, 0)
	}

	return r, nil
}

// ListMyGames lists the games one develops (ie. can edit)
func (c *Client) ListMyGames() (r ListMyGamesResponse, err error) {
	path := c.MakePath("my-games")

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

// GameUploadsResponse is what the server replies with when asked for a game's uploads
type GameUploadsResponse struct {
	Response

	Uploads []*Upload `json:"uploads"`
}

// GameUploads lists the uploads for a game that we have access to with our API key
func (c *Client) GameUploads(gameID int64) (r GameUploadsResponse, err error) {
	path := c.MakePath("game/%d/uploads", gameID)

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

// UploadDownloadResponse is what the API replies to when we ask to download an upload
type UploadDownloadResponse struct {
	Response

	URL string
}

// UploadDownload attempts to download an upload without a download key
func (c *Client) UploadDownload(uploadID int64) (r UploadDownloadResponse, err error) {
	return c.UploadDownloadWithKey("", uploadID)
}

// UploadDownloadWithKey attempts to download an upload with a download key
func (c *Client) UploadDownloadWithKey(downloadKey string, uploadID int64) (r UploadDownloadResponse, err error) {
	return c.UploadDownloadWithKeyAndValues(downloadKey, uploadID, nil)
}

// UploadDownloadWithKeyAndValues attempts to download an upload with a download key and additional parameters
func (c *Client) UploadDownloadWithKeyAndValues(downloadKey string, uploadID int64, values url.Values) (r UploadDownloadResponse, err error) {
	if values == nil {
		values = url.Values{}
	}

	if downloadKey != "" {
		values.Add("download_key_id", downloadKey)
	}

	path := c.MakePath("upload/%d/download", uploadID)
	if len(values) > 0 {
		path = fmt.Sprintf("%s?%s", path, values.Encode())
	}

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return r, err
}

// NewBuildResponse is what the API replies with when we create a new build
type NewBuildResponse struct {
	Response

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
func (c *Client) CreateBuild(target string, channel string, userVersion string) (r NewBuildResponse, err error) {
	path := c.MakePath("wharf/builds")

	form := url.Values{}
	form.Add("target", target)
	form.Add("channel", channel)
	if userVersion != "" {
		form.Add("user_version", userVersion)
	}

	resp, err := c.PostForm(path, form)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

// ListChannels returns a list of the channels for a game
func (c *Client) ListChannels(target string) (r ListChannelsResponse, err error) {
	form := url.Values{}
	form.Add("target", target)
	path := c.MakePath("wharf/channels?%s", form.Encode())

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

// GetChannel returns information about a given channel for a given game
func (c *Client) GetChannel(target string, channel string) (r GetChannelResponse, err error) {
	form := url.Values{}
	form.Add("target", target)
	path := c.MakePath("wharf/channels/%s?%s", channel, form.Encode())

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

// BuildFileType describes the type of a build file: patch, archive, signature, etc.
type BuildFileType string

const (
	// BuildFileTypePatch describes wharf patch files (.pwr)
	BuildFileTypePatch BuildFileType = "patch"
	// BuildFileTypeArchive describes canonical archive form (.zip)
	BuildFileTypeArchive = "archive"
	// BuildFileTypeSignature describes wharf signature files (.pws)
	BuildFileTypeSignature = "signature"
	// BuildFileTypeManifest is reserved
	BuildFileTypeManifest = "manifest"
	// BuildFileTypeUnpacked describes the single file that is in the build (if it was just a single file)
	BuildFileTypeUnpacked = "unpacked"
)

// BuildFileSubType describes the subtype of a build file: mostly its compression
// level. For example, rediff'd patches are "optimized", whereas initial patches are "default"
type BuildFileSubType string

const (
	// BuildFileSubTypeDefault describes default compression (rsync patches)
	BuildFileSubTypeDefault BuildFileSubType = "default"
	// BuildFileSubTypeGzip is reserved
	BuildFileSubTypeGzip = "gzip"
	// BuildFileSubTypeOptimized describes optimized compression (rediff'd / bsdiff patches)
	BuildFileSubTypeOptimized = "optimized"
)

// UploadType describes which strategy is used for uploading to storage
// some types allow for uploading in blocks (which is resumable), some
// expect the whole payload in one request.
type UploadType string

const (
	// UploadTypeMultipart lets you send metadata + all the content in a single request
	UploadTypeMultipart UploadType = "multipart"
	// UploadTypeResumable lets you send blocks of N*128KB at a time. The upload session is
	// started from the API server, so the ingest point will be anchored wherever the API server is.
	UploadTypeResumable = "resumable"
	// UploadTypeDeferredResumable also lets you send blocks of N*128KB at a time, but it
	// lets you start the upload session from the client, which means you might get a closer ingest point.
	UploadTypeDeferredResumable = "deferred_resumable"
)

// BuildState describes the state of a build, relative to its initial upload, and
// its processing.
type BuildState string

const (
	// BuildStateStarted is the state of a build from its creation until the initial upload is complete
	BuildStateStarted BuildState = "started"
	// BuildStateProcessing is the state of a build from the initial upload's completion to its fully-processed state.
	// This state does not mean the build is actually being processed right now, it's just queued for processing.
	BuildStateProcessing = "processing"
	// BuildStateCompleted means the build was successfully processed. Its patch hasn't necessarily been
	// rediff'd yet, but we have the holy (patch,signature,archive) trinity.
	BuildStateCompleted = "completed"
	// BuildStateFailed means something went wrong with the build. A failing build will not update the channel
	// head and can be requeued by the itch.io team, although if a new build is pushed before they do,
	// that new build will "win".
	BuildStateFailed = "failed"
)

// BuildFileState describes the state of a specific file for a build
type BuildFileState string

const (
	// BuildFileStateCreated means the file entry exists on itch.io
	BuildFileStateCreated BuildFileState = "created"
	// BuildFileStateUploading means the file is currently being uploaded to storage
	BuildFileStateUploading = "uploading"
	// BuildFileStateUploaded means the file is ready
	BuildFileStateUploaded = "uploaded"
	// BuildFileStateFailed means the file failed uploading
	BuildFileStateFailed = "failed"
)

// ListBuildFiles returns a list of files associated to a build
func (c *Client) ListBuildFiles(buildID int64) (r ListBuildFilesResponse, err error) {
	path := c.MakePath("wharf/builds/%d/files", buildID)

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

// CreateBuildFile creates a new build file for a build
func (c *Client) CreateBuildFile(buildID int64, fileType BuildFileType, subType BuildFileSubType, uploadType UploadType) (r CreateBuildFileResponse, err error) {
	path := c.MakePath("wharf/builds/%d/files", buildID)

	form := url.Values{}
	form.Add("type", string(fileType))
	if subType != "" {
		form.Add("sub_type", string(subType))
	}
	if uploadType != "" {
		form.Add("upload_type", string(uploadType))
	}

	resp, err := c.PostForm(path, form)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

// CreateBuildFileWithName creates a new build file for a build, with a specific name
func (c *Client) CreateBuildFileWithName(buildID int64, fileType BuildFileType, subType BuildFileSubType, uploadType UploadType, name string) (r CreateBuildFileResponse, err error) {
	path := c.MakePath("wharf/builds/%d/files", buildID)

	form := url.Values{}
	form.Add("type", string(fileType))
	if subType != "" {
		form.Add("sub_type", string(subType))
	}
	if uploadType != "" {
		form.Add("upload_type", string(uploadType))
	}
	if name != "" {
		form.Add("filename", name)
	}

	resp, err := c.PostForm(path, form)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

// FinalizeBuildFile marks the end of the upload for a build file, it validates
func (c *Client) FinalizeBuildFile(buildID int64, fileID int64, size int64) (r FinalizeBuildFileResponse, err error) {
	path := c.MakePath("wharf/builds/%d/files/%d", buildID, fileID)

	form := url.Values{}
	form.Add("size", fmt.Sprintf("%d", size))

	resp, err := c.PostForm(path, form)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

// DownloadBuildFileResponse is what the API responds with when we
// ask to download an upload
type DownloadBuildFileResponse struct {
	Response

	URL string
}

var (
	// ErrBuildFileNotFound is returned when someone is asking for a non-existent file
	ErrBuildFileNotFound = errors.New("build file not found in storage")
)

// GetBuildFileDownloadURL returns a download URL for a given build file
func (c *Client) GetBuildFileDownloadURL(buildID int64, fileID int64) (r DownloadBuildFileResponse, err error) {
	return c.GetBuildFileDownloadURLWithValues(buildID, fileID, nil)
}

// GetBuildFileDownloadURLWithValues returns a download URL for a given build file, with additional query parameters
func (c *Client) GetBuildFileDownloadURLWithValues(buildID int64, fileID int64, values url.Values) (r DownloadBuildFileResponse, err error) {
	path := c.MakePath("wharf/builds/%d/files/%d/download", buildID, fileID)
	if values != nil {
		path = path + "?" + values.Encode()
	}

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	if err != nil {
		return
	}

	return
}

// DownloadBuildFile returns an io.ReadCloser to download a build file, as
// opposed to GetBuildFileDownloadURL
func (c *Client) DownloadBuildFile(buildID int64, fileID int64) (reader io.ReadCloser, err error) {
	path := c.MakePath("wharf/builds/%d/files/%d/download", buildID, fileID)

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	var r DownloadBuildFileResponse
	err = ParseAPIResponse(&r, resp)
	if err != nil {
		return
	}

	req, err := http.NewRequest("GET", r.URL, nil)
	if err != nil {
		return
	}

	// not an API request, going directly with http's DefaultClient
	dlResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	if dlResp.StatusCode == 200 {
		reader = dlResp.Body
		return
	}

	dlResp.Body.Close()

	if dlResp.StatusCode == 404 {
		err = ErrBuildFileNotFound
	} else {
		err = fmt.Errorf("Can't download: %s", dlResp.Status)
	}
	return
}

// DownloadUploadBuild returns download info for all types of files for a build
func (c *Client) DownloadUploadBuild(uploadID int64, buildID int64) (r DownloadUploadBuildResponse, err error) {
	return c.DownloadUploadBuildWithKey("", uploadID, buildID)
}

// DownloadUploadBuildWithKey returns download info for all types of files for a build, when using with a download key
func (c *Client) DownloadUploadBuildWithKey(downloadKey string, uploadID int64, buildID int64) (r DownloadUploadBuildResponse, err error) {
	return c.DownloadUploadBuildWithKeyAndValues(downloadKey, uploadID, buildID, nil)
}

// DownloadUploadBuildWithKeyAndValues returns download info for all types of files for a build.
// downloadKey can be empty
func (c *Client) DownloadUploadBuildWithKeyAndValues(downloadKey string, uploadID int64, buildID int64, values url.Values) (r DownloadUploadBuildResponse, err error) {
	if values == nil {
		values = url.Values{}
	}

	if downloadKey != "" {
		values.Add("download_key_id", downloadKey)
	}

	path := c.MakePath("upload/%d/download/builds/%d", uploadID, buildID)
	if len(values) > 0 {
		path = fmt.Sprintf("%s?%s", path, values.Encode())
	}

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	if err != nil {
		return
	}

	return
}

// BuildEventType specifies what kind of event a build event is - could be a log message, etc.
type BuildEventType string

const (
	// BuildEventLog is for build events of type log message
	BuildEventLog BuildEventType = "log"
)

// BuildEventData is a JSON object associated with a build event
type BuildEventData map[string]interface{}

// CreateBuildEvent associates a new build event to a build
func (c *Client) CreateBuildEvent(buildID int64, eventType BuildEventType, message string, data BuildEventData) (r CreateBuildEventResponse, err error) {
	path := c.MakePath("wharf/builds/%d/events", buildID)

	form := url.Values{}
	form.Add("type", string(eventType))
	form.Add("message", message)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	form.Add("data", string(jsonData))

	resp, err := c.PostForm(path, form)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

// CreateBuildFailure marks a given build as failed. We get to specify an error message and
// if it's a fatal error (if not, the build can be retried after a bit)
func (c *Client) CreateBuildFailure(buildID int64, message string, fatal bool) (r CreateBuildFailureResponse, err error) {
	path := c.MakePath("wharf/builds/%d/failures", buildID)

	form := url.Values{}
	form.Add("message", message)
	if fatal {
		form.Add("fatal", fmt.Sprintf("%v", fatal))
	}

	resp, err := c.PostForm(path, form)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

// CreateRediffBuildFailure marks a given build as having failed to rediff (optimize)
func (c *Client) CreateRediffBuildFailure(buildID int64, message string) (r CreateBuildFailureResponse, err error) {
	path := c.MakePath("wharf/builds/%d/failures/rediff", buildID)

	form := url.Values{}
	form.Add("message", message)

	resp, err := c.PostForm(path, form)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

// ListBuildEvents returns a series of events associated with a given build
func (c *Client) ListBuildEvents(buildID int64) (r ListBuildEventsResponse, err error) {
	path := c.MakePath("wharf/builds/%d/events", buildID)

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

type ListGameUploadsParams struct {
	GameID        int64
	DownloadKeyID int64
	ExtraQuery    url.Values
}

// ListGameUploads
func (c *Client) ListGameUploads(params *ListGameUploadsParams) (ListGameUploadsResponse, error) {
	var r ListGameUploadsResponse

	if params.GameID == 0 {
		return r, errors.New("Missing GameID")
	}

	path := c.MakePath("/game/%d/uploads", params.GameID)
	if params.DownloadKeyID != 0 {
		path = c.MakePath("/download-key/%d/uploads", params.DownloadKeyID)
	}

	if params.ExtraQuery != nil {
		path = fmt.Sprintf("%s?%s", path, params.ExtraQuery.Encode())
	}

	resp, err := c.Get(path)
	if err != nil {
		return r, errors.Wrap(err, 0)
	}

	err = ParseAPIResponse(&r, resp)
	if err != nil {
		return r, errors.Wrap(err, 0)
	}
	return r, nil
}

type FindUpgradeParams struct {
	UploadID       int64
	CurrentBuildID int64
	DownloadKeyID  int64
}

// FindUpgrade
func (c *Client) FindUpgrade(params *FindUpgradeParams) (FindUpgradeResponse, error) {
	var r FindUpgradeResponse

	if params.UploadID == 0 {
		return r, errors.New("Missing UploadID")
	}

	if params.CurrentBuildID == 0 {
		return r, errors.New("Missing CurrentBuildID")
	}

	form := url.Values{}
	form.Add("v", "2")

	if params.DownloadKeyID != 0 {
		form.Add("download_key_id", fmt.Sprintf("%d", params.DownloadKeyID))
	}

	path := c.MakePath("/upload/%d/upgrade/%d?%s", params.UploadID, params.CurrentBuildID, form.Encode())

	resp, err := c.Get(path)
	if err != nil {
		return r, errors.Wrap(err, 0)
	}

	err = ParseAPIResponse(&r, resp)
	if err != nil {
		return r, errors.Wrap(err, 0)
	}
	return r, nil
}
