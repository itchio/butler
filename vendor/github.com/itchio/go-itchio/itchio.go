package itchio

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

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

func (c *Client) SetServer(itchioServer string) *Client {
	c.BaseURL = fmt.Sprintf("%s/api/1", itchioServer)
	return c
}

type Response struct {
	Errors []string
}

type User struct {
	ID       int64
	Username string
	CoverUrl string `json:"cover_url"`
}

type Game struct {
	ID  int64
	Url string
}

type Upload struct {
	ID       int64
	Filename string
	Size     int64

	OSX     bool `json:"p_osx"`
	Linux   bool `json:"p_linux"`
	Windows bool `json:"p_windows"`
	Android bool `json:"p_android"`
}

type StatusResponse struct {
	Response

	Success bool
}

func (c *Client) WharfStatus() (r StatusResponse, err error) {
	path := c.MakePath("wharf/status")

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

type MyGamesResponse struct {
	Response

	Games []Game
}

func (c *Client) MyGames() (r MyGamesResponse, err error) {
	path := c.MakePath("my-games")
	log.Printf("Requesting %s\n", path)

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

type GameUploadsResponse struct {
	Response

	Uploads []Upload `json:"uploads"`
}

func (c *Client) GameUploads(gameID int64) (r GameUploadsResponse, err error) {
	path := c.MakePath("game/%d/uploads", gameID)

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

type UploadDownloadResponse struct {
	Response

	URL string
}

func (c *Client) UploadDownload(uploadID int64) (r UploadDownloadResponse, err error) {
	path := c.MakePath("upload/%d/download", uploadID)

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return r, err
}

type NewBuildResponse struct {
	Response

	Build struct {
		ID          int64 `json:"id"`
		UploadID    int64 `json:"upload_id"`
		ParentBuild struct {
			ID int64 `json:"id"`
		} `json:"parent_build"`
	}
}

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

type BuildFileInfo struct {
	ID      int64
	Size    int64
	State   BuildFileState
	Type    BuildFileType    `json:"type"`
	SubType BuildFileSubType `json:"sub_type"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type BuildInfo struct {
	ID            int64
	ParentBuildID int64 `json:"parent_build_id"`
	State         BuildState

	Files []*BuildFileInfo

	User      User
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ChannelInfo struct {
	Name string
	Tags string

	Upload  Upload
	Head    *BuildInfo `json:"head"`
	Pending *BuildInfo `json:"pending"`
}

type ListChannelsResponse struct {
	Response

	Channels map[string]ChannelInfo
}

type GetChannelResponse struct {
	Response

	Channel ChannelInfo
}

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

type BuildFileType string

const (
	BuildFileType_PATCH     BuildFileType = "patch"
	BuildFileType_ARCHIVE                 = "archive"
	BuildFileType_SIGNATURE               = "signature"
	BuildFileType_MANIFEST                = "manifest"
)

type BuildFileSubType string

const (
	BuildFileSubType_DEFAULT   BuildFileSubType = "default"
	BuildFileSubType_GZIP                       = "gzip"
	BuildFileSubType_OPTIMIZED                  = "optimized"
)

type UploadType string

const (
	UploadType_MULTIPART          UploadType = "multipart"
	UploadType_RESUMABLE                     = "resumable"
	UploadType_DEFERRED_RESUMABLE            = "deferred_resumable"
)

type BuildState string

const (
	BuildState_STARTED    BuildState = "started"
	BuildState_PROCESSING            = "processing"
	BuildState_COMPLETED             = "completed"
	BuildState_FAILED                = "failed"
)

type BuildFileState string

const (
	BuildFileState_CREATED   BuildFileState = "created"
	BuildFileState_UPLOADING                = "uploading"
	BuildFileState_UPLOADED                 = "uploaded"
	BuildFileState_FAILED                   = "failed"
)

type ListBuildFilesResponse struct {
	Response

	Files []*BuildFileInfo
}

func (c *Client) ListBuildFiles(buildID int64) (r ListBuildFilesResponse, err error) {
	path := c.MakePath("wharf/builds/%d/files", buildID)

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

type NewBuildFileResponse struct {
	Response

	File struct {
		ID            int64
		UploadURL     string            `json:"upload_url"`
		UploadParams  map[string]string `json:"upload_params"`
		UploadHeaders map[string]string `json:"upload_headers"`
	}
}

func (c *Client) CreateBuildFile(buildID int64, fileType BuildFileType, subType BuildFileSubType, uploadType UploadType) (r NewBuildFileResponse, err error) {
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

type FinalizeBuildFileResponse struct {
	Response
}

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

type DownloadBuildFileResponse struct {
	Response

	URL string
}

var (
	BuildFileNotFound = errors.New("build file not found in storage")
)

func (c *Client) GetBuildFileDownloadURL(buildID int64, fileID int64) (r DownloadBuildFileResponse, err error) {
	path := c.MakePath("wharf/builds/%d/files/%d/download", buildID, fileID)

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
		err = BuildFileNotFound
	} else {
		err = fmt.Errorf("Can't download: %s", dlResp.Status)
	}
	return
}

type DownloadUploadBuildResponseItem struct {
	URL string
}

type DownloadUploadBuildResponse struct {
	Response

	Patch     *DownloadUploadBuildResponseItem
	Signature *DownloadUploadBuildResponseItem
	Manifest  *DownloadUploadBuildResponseItem
	Archive   *DownloadUploadBuildResponseItem
}

func (c *Client) DownloadUploadBuild(uploadID int64, buildID int64) (r DownloadUploadBuildResponse, err error) {
	path := c.MakePath("/upload/%d/download/builds/%d", uploadID, buildID)

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

func (c *Client) DownloadUploadBuildWithKey(downloadKey string, uploadID int64, buildID int64) (r DownloadUploadBuildResponse, err error) {
	path := c.MakePath("/download-key/%s/download/%d/builds/%d", downloadKey, uploadID, buildID)

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

type BuildEventType string

const (
	BuildEvent_JOB_STARTED   BuildEventType = "job_started"
	BuildEvent_JOB_FAILED                   = "job_failed"
	BuildEvent_JOB_COMPLETED                = "job_completed"
)

type BuildEventData map[string]interface{}

type NewBuildEventResponse struct {
	Response
}

func (c *Client) CreateBuildEvent(buildID int64, eventType BuildEventType, message string, data BuildEventData) (r NewBuildEventResponse, err error) {
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

type CreateBuildFailureResponse struct {
	Response
}

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

type BuildEvent struct {
	Type    BuildEventType
	Message string
	Data    BuildEventData
}

type ListBuildEventsResponse struct {
	Response

	Events []BuildEvent
}

func (c *Client) ListBuildEvents(buildID int64) (r ListBuildEventsResponse, err error) {
	path := c.MakePath("wharf/builds/%d/events", buildID)

	resp, err := c.Get(path)
	if err != nil {
		return
	}

	err = ParseAPIResponse(&r, resp)
	return
}

// Helpers

func (c *Client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *Client) PostForm(url string, data url.Values) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.Do(req)
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(c.Key, "jwt:") {
		req.Header.Add("Authorization", strings.Split(c.Key, ":")[1])
	}
	req.Header.Set("User-Agent", c.UserAgent)

	var res *http.Response
	var err error

	retryPatterns := append(c.RetryPatterns, time.Millisecond)

	for _, sleepTime := range retryPatterns {
		res, err = c.HTTPClient.Do(req)
		if err != nil {
			if strings.Contains(err.Error(), "TLS handshake timeout") {
				time.Sleep(sleepTime + time.Duration(rand.Int()%1000)*time.Millisecond)
				continue
			}
			return nil, err
		}

		if res.StatusCode == 503 {
			// Rate limited, try again according to patterns.
			// following https://cloud.google.com/storage/docs/json_api/v1/how-tos/upload#exp-backoff to the letter
			res.Body.Close()
			time.Sleep(sleepTime + time.Duration(rand.Int()%1000)*time.Millisecond)
			continue
		}

		break
	}

	return res, err
}

func (c *Client) MakePath(format string, a ...interface{}) string {
	base := strings.Trim(c.BaseURL, "/")
	subPath := strings.Trim(fmt.Sprintf(format, a...), "/")

	var key string
	if strings.HasPrefix(c.Key, "jwt:") {
		key = "jwt"
	} else {
		key = c.Key
	}
	return fmt.Sprintf("%s/%s/%s", base, key, subPath)
}

func ParseAPIResponse(dst interface{}, res *http.Response) error {
	if res == nil || res.Body == nil {
		return fmt.Errorf("No response from server")
	}

	bodyReader := res.Body
	defer bodyReader.Close()

	if res.StatusCode/100 != 2 {
		return fmt.Errorf("Server returned %s for %s", res.Status, res.Request.URL.String())
	}

	err := json.NewDecoder(bodyReader).Decode(dst)
	if err != nil {
		return err
	}

	errs := reflect.Indirect(reflect.ValueOf(dst)).FieldByName("Errors")
	if errs.Len() > 0 {
		// TODO: handle other errors too
		return fmt.Errorf("itch.io API error: %s", errs.Index(0).String())
	}

	return nil
}

func FindBuildFile(fileType BuildFileType, files []*BuildFileInfo) *BuildFileInfo {
	for _, f := range files {
		if f.Type == fileType && f.State == BuildFileState_UPLOADED {
			return f
		}
	}

	return nil
}
