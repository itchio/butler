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
func (c *Client) WharfStatus() (*WharfStatusResponse, error) {
	r := &WharfStatusResponse{}

	err := c.GetResponse(c.MakePath("wharf/status"), r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// GetMe returns information about to which the current credentials belong
func (c *Client) GetMe() (*GetMeResponse, error) {
	r := &GetMeResponse{}

	err := c.GetResponse(c.MakePath("me"), r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

type GetGameParams struct {
	GameID int64 `json:"gameId"`
}

func (c *Client) GetGame(params *GetGameParams) (*GetGameResponse, error) {
	r := &GetGameResponse{}

	path := c.MakePath("game/%d", params.GameID)
	err := c.GetResponse(path, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

type GetCollectionParams struct {
	CollectionID int64 `json:"collectionId"`
}

func (c *Client) GetCollection(params *GetCollectionParams) (*GetCollectionResponse, error) {
	r := &GetCollectionResponse{}

	path := c.MakePath("collection/%d", params.CollectionID)
	err := c.GetResponse(path, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

type GetCollectionGamesParams struct {
	CollectionID int64 `json:"collectionId"`
	Page         int64 `json:"page"`
}

func (c *Client) GetCollectionGames(params *GetCollectionGamesParams) (*GetCollectionGamesResponse, error) {
	r := &GetCollectionGamesResponse{}

	values := url.Values{}
	values.Add("page", fmt.Sprintf("%d", params.Page))

	path := c.MakePath("collection/%d/games?%s", params.CollectionID, values.Encode())
	err := c.GetResponse(path, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// ListMyGames lists the games one develops (ie. can edit)
func (c *Client) ListMyGames() (*ListMyGamesResponse, error) {
	r := &ListMyGamesResponse{}

	err := c.GetResponse(c.MakePath("my-games"), r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// ListMyOwnedKeys lists the download keys one owns
func (c *Client) ListMyOwnedKeys() (*ListMyOwnedKeysResponse, error) {
	r := &ListMyOwnedKeysResponse{}

	err := c.GetResponse(c.MakePath("my-owned-keys"), r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// ListMyCollections lists the collections associated to a profile
func (c *Client) ListMyCollections() (*ListMyCollectionsResponse, error) {
	r := &ListMyCollectionsResponse{}

	err := c.GetResponse(c.MakePath("my-collections"), r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// GameUploads lists the uploads for a game that we have access to with our API key
func (c *Client) GameUploads(gameID int64) (*GameUploadsResponse, error) {
	r := &GameUploadsResponse{}
	path := c.MakePath("game/%d/uploads", gameID)

	err := c.GetResponse(path, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// UploadDownload attempts to download an upload without a download key
func (c *Client) UploadDownload(uploadID int64) (*UploadDownloadResponse, error) {
	return c.UploadDownloadWithKey("", uploadID)
}

// UploadDownloadWithKey attempts to download an upload with a download key
func (c *Client) UploadDownloadWithKey(downloadKey string, uploadID int64) (*UploadDownloadResponse, error) {
	return c.UploadDownloadWithKeyAndValues(downloadKey, uploadID, nil)
}

// UploadDownloadWithKeyAndValues attempts to download an upload with a download key and additional parameters
func (c *Client) UploadDownloadWithKeyAndValues(downloadKey string, uploadID int64, values url.Values) (*UploadDownloadResponse, error) {
	r := &UploadDownloadResponse{}
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

	err := c.GetResponse(path, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// CreateBuild creates a new build for a given user/game:channel, with
// an optional user version
func (c *Client) CreateBuild(target string, channel string, userVersion string) (*NewBuildResponse, error) {
	r := &NewBuildResponse{}
	path := c.MakePath("wharf/builds")

	form := url.Values{}
	form.Add("target", target)
	form.Add("channel", channel)
	if userVersion != "" {
		form.Add("user_version", userVersion)
	}

	err := c.PostFormResponse(path, form, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// ListChannels returns a list of the channels for a game
func (c *Client) ListChannels(target string) (*ListChannelsResponse, error) {
	r := &ListChannelsResponse{}
	form := url.Values{}
	form.Add("target", target)
	path := c.MakePath("wharf/channels?%s", form.Encode())

	err := c.GetResponse(path, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// GetChannel returns information about a given channel for a given game
func (c *Client) GetChannel(target string, channel string) (*GetChannelResponse, error) {
	r := &GetChannelResponse{}
	form := url.Values{}
	form.Add("target", target)
	path := c.MakePath("wharf/channels/%s?%s", channel, form.Encode())

	err := c.GetResponse(path, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

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

// ListBuildFiles returns a list of files associated to a build
func (c *Client) ListBuildFiles(buildID int64) (*ListBuildFilesResponse, error) {
	r := &ListBuildFilesResponse{}
	path := c.MakePath("wharf/builds/%d/files", buildID)

	err := c.GetResponse(path, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// CreateBuildFile creates a new build file for a build
func (c *Client) CreateBuildFile(buildID int64, fileType BuildFileType, subType BuildFileSubType, uploadType UploadType) (*CreateBuildFileResponse, error) {
	r := &CreateBuildFileResponse{}
	path := c.MakePath("wharf/builds/%d/files", buildID)

	form := url.Values{}
	form.Add("type", string(fileType))
	if subType != "" {
		form.Add("sub_type", string(subType))
	}
	if uploadType != "" {
		form.Add("upload_type", string(uploadType))
	}

	err := c.PostFormResponse(path, form, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// CreateBuildFileWithName creates a new build file for a build, with a specific name
func (c *Client) CreateBuildFileWithName(buildID int64, fileType BuildFileType, subType BuildFileSubType, uploadType UploadType, name string) (*CreateBuildFileResponse, error) {
	r := &CreateBuildFileResponse{}
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

	err := c.PostFormResponse(path, form, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// FinalizeBuildFile marks the end of the upload for a build file, it validates
func (c *Client) FinalizeBuildFile(buildID int64, fileID int64, size int64) (*FinalizeBuildFileResponse, error) {
	r := &FinalizeBuildFileResponse{}
	path := c.MakePath("wharf/builds/%d/files/%d", buildID, fileID)

	form := url.Values{}
	form.Add("size", fmt.Sprintf("%d", size))

	err := c.PostFormResponse(path, form, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

var (
	// ErrBuildFileNotFound is returned when someone is asking for a non-existent file
	ErrBuildFileNotFound = errors.New("build file not found in storage")
)

// GetBuildFileDownloadURL returns a download URL for a given build file
func (c *Client) GetBuildFileDownloadURL(buildID int64, fileID int64) (*DownloadBuildFileResponse, error) {
	return c.GetBuildFileDownloadURLWithValues(buildID, fileID, nil)
}

// GetBuildFileDownloadURLWithValues returns a download URL for a given build file, with additional query parameters
func (c *Client) GetBuildFileDownloadURLWithValues(buildID int64, fileID int64, values url.Values) (*DownloadBuildFileResponse, error) {
	r := &DownloadBuildFileResponse{}
	path := c.MakePath("wharf/builds/%d/files/%d/download", buildID, fileID)
	if values != nil {
		path = path + "?" + values.Encode()
	}

	err := c.GetResponse(path, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// DownloadBuildFile returns an io.ReadCloser to download a build file, as
// opposed to GetBuildFileDownloadURL
func (c *Client) DownloadBuildFile(buildID int64, fileID int64) (io.ReadCloser, error) {
	path := c.MakePath("wharf/builds/%d/files/%d/download", buildID, fileID)

	r := &DownloadBuildFileResponse{}
	err := c.GetResponse(path, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	req, err := http.NewRequest("GET", r.URL, nil)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	// not an API request, going directly with http's DefaultClient
	dlResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if dlResp.StatusCode == 200 {
		return dlResp.Body, nil
	}

	// non-200 status, emit an error
	dlResp.Body.Close()

	if dlResp.StatusCode == 404 {
		return nil, ErrBuildFileNotFound
	}
	return nil, fmt.Errorf("Can't download: %s", dlResp.Status)
}

// DownloadUploadBuild returns download info for all types of files for a build
func (c *Client) DownloadUploadBuild(uploadID int64, buildID int64) (*DownloadUploadBuildResponse, error) {
	return c.DownloadUploadBuildWithKey("", uploadID, buildID)
}

// DownloadUploadBuildWithKey returns download info for all types of files for a build, when using with a download key
func (c *Client) DownloadUploadBuildWithKey(downloadKey string, uploadID int64, buildID int64) (*DownloadUploadBuildResponse, error) {
	return c.DownloadUploadBuildWithKeyAndValues(downloadKey, uploadID, buildID, nil)
}

// DownloadUploadBuildWithKeyAndValues returns download info for all types of files for a build.
// downloadKey can be empty
func (c *Client) DownloadUploadBuildWithKeyAndValues(downloadKey string, uploadID int64, buildID int64, values url.Values) (*DownloadUploadBuildResponse, error) {
	r := &DownloadUploadBuildResponse{}
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

	err := c.GetResponse(path, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
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
func (c *Client) CreateBuildEvent(buildID int64, eventType BuildEventType, message string, data BuildEventData) (*CreateBuildEventResponse, error) {
	r := &CreateBuildEventResponse{}
	path := c.MakePath("wharf/builds/%d/events", buildID)

	form := url.Values{}
	form.Add("type", string(eventType))
	form.Add("message", message)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	form.Add("data", string(jsonData))

	err = c.PostFormResponse(path, form, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// CreateBuildFailure marks a given build as failed. We get to specify an error message and
// if it's a fatal error (if not, the build can be retried after a bit)
func (c *Client) CreateBuildFailure(buildID int64, message string, fatal bool) (*CreateBuildFailureResponse, error) {
	r := &CreateBuildFailureResponse{}
	path := c.MakePath("wharf/builds/%d/failures", buildID)

	form := url.Values{}
	form.Add("message", message)
	if fatal {
		form.Add("fatal", fmt.Sprintf("%v", fatal))
	}

	err := c.PostFormResponse(path, form, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// CreateRediffBuildFailure marks a given build as having failed to rediff (optimize)
func (c *Client) CreateRediffBuildFailure(buildID int64, message string) (*CreateBuildFailureResponse, error) {
	r := &CreateBuildFailureResponse{}
	path := c.MakePath("wharf/builds/%d/failures/rediff", buildID)

	form := url.Values{}
	form.Add("message", message)

	err := c.PostFormResponse(path, form, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// ListBuildEvents returns a series of events associated with a given build
func (c *Client) ListBuildEvents(buildID int64) (*ListBuildEventsResponse, error) {
	r := &ListBuildEventsResponse{}
	path := c.MakePath("wharf/builds/%d/events", buildID)

	err := c.GetResponse(path, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

type ListGameUploadsParams struct {
	GameID        int64
	DownloadKeyID int64
	ExtraQuery    url.Values
}

// ListGameUploads
func (c *Client) ListGameUploads(params *ListGameUploadsParams) (*ListGameUploadsResponse, error) {
	r := &ListGameUploadsResponse{}

	if params.GameID == 0 {
		return nil, errors.New("Missing GameID")
	}

	path := c.MakePath("/game/%d/uploads", params.GameID)
	if params.DownloadKeyID != 0 {
		path = c.MakePath("/download-key/%d/uploads", params.DownloadKeyID)
	}

	if params.ExtraQuery != nil {
		path = fmt.Sprintf("%s?%s", path, params.ExtraQuery.Encode())
	}

	err := c.GetResponse(path, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

type FindUpgradeParams struct {
	UploadID       int64
	CurrentBuildID int64
	DownloadKeyID  int64
}

// FindUpgrade looks for a series of patch to upgrade from a given version to the latest version
func (c *Client) FindUpgrade(params *FindUpgradeParams) (*FindUpgradeResponse, error) {
	r := &FindUpgradeResponse{}

	if params.UploadID == 0 {
		return nil, errors.New("Missing UploadID")
	}

	if params.CurrentBuildID == 0 {
		return nil, errors.New("Missing CurrentBuildID")
	}

	form := url.Values{}
	form.Add("v", "2")

	if params.DownloadKeyID != 0 {
		form.Add("download_key_id", fmt.Sprintf("%d", params.DownloadKeyID))
	}

	path := c.MakePath("/upload/%d/upgrade/%d?%s", params.UploadID, params.CurrentBuildID, form.Encode())

	err := c.GetResponse(path, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

type NewDownloadSessionParams struct {
	GameID        int64
	DownloadKeyID int64
}

func (c *Client) NewDownloadSession(params *NewDownloadSessionParams) (*NewDownloadSessionResponse, error) {
	r := &NewDownloadSessionResponse{}
	path := c.MakePath("/game/%d/download", params.GameID)

	form := url.Values{}
	if params.DownloadKeyID != 0 {
		form.Add("download_key_id", fmt.Sprintf("%d", params.DownloadKeyID))
	}

	err := c.PostFormResponse(path, form, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

type LoginWithPasswordParams struct {
	Username          string
	Password          string
	RecaptchaResponse string
}

func (c *Client) LoginWithPassword(params *LoginWithPasswordParams) (*LoginWithPasswordResponse, error) {
	r := &LoginWithPasswordResponse{}
	path := c.MakeRootPath("/login")

	form := url.Values{}
	form.Add("v", "3")
	form.Add("source", "desktop")
	form.Add("username", params.Username)
	form.Add("password", params.Password)
	if params.RecaptchaResponse != "" {
		form.Add("recaptcha_response", params.RecaptchaResponse)
	}

	err := c.PostFormResponse(path, form, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

type TOTPVerifyParams struct {
	Token string
	Code  string
}

func (c *Client) TOTPVerify(params *TOTPVerifyParams) (*TOTPVerifyResponse, error) {
	r := &TOTPVerifyResponse{}
	path := c.MakeRootPath("/totp/verify")

	form := url.Values{}
	form.Add("token", params.Token)
	form.Add("code", params.Code)

	err := c.PostFormResponse(path, form, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

type SubkeyParams struct {
	GameID int64
	Scope  string
}

func (c *Client) Subkey(params *SubkeyParams) (*SubkeyResponse, error) {
	r := &SubkeyResponse{}
	path := c.MakePath("/credentials/subkey")

	form := url.Values{}
	form.Add("game_id", fmt.Sprintf("%d", params.GameID))
	form.Add("scope", params.Scope)

	err := c.PostFormResponse(path, form, r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r, nil
}

// Dates

const APIDateFormat = "2006-01-02 15:04:05"
