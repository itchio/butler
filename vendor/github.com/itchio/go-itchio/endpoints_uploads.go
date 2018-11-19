package itchio

// GameCredentials is your one-stop shop for all the
// things that allow access to a game or its uploads, such as:
// a download key, a password (for restricted pages), a secret
// (for private pages).
type GameCredentials struct {
	DownloadKeyID int64  `json:"downloadKeyId,omitempty"`
	Password      string `json:"password,omitempty"`
	Secret        string `json:"secret,omitempty"`
}

//-------------------------------------------------------

// ListGameUploadsParams : params for ListGameUploads
type ListGameUploadsParams struct {
	GameID int64

	// Optional
	Credentials GameCredentials
}

// ListGameUploadsResponse : response
type ListGameUploadsResponse struct {
	Uploads []*Upload `json:"uploads"`
}

// ListGameUploads lists the uploads for a game that we have access to with our API key
// and game credentials.
func (c *Client) ListGameUploads(p ListGameUploadsParams) (*ListGameUploadsResponse, error) {
	q := NewQuery(c, "/games/%d/uploads", p.GameID)
	q.AddGameCredentials(p.Credentials)
	r := &ListGameUploadsResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

// GetUploadParams : params for GetUpload
type GetUploadParams struct {
	UploadID int64

	// Optional
	Credentials GameCredentials
}

// GetUploadResponse : response for GetUpload
type GetUploadResponse struct {
	Upload *Upload `json:"upload"`
}

// GetUpload retrieves information about a single upload, by ID.
func (c *Client) GetUpload(params GetUploadParams) (*GetUploadResponse, error) {
	q := NewQuery(c, "/uploads/%d", params.UploadID)
	q.AddGameCredentials(params.Credentials)
	r := &GetUploadResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

// ListUploadBuildsParams : params for ListUploadBuilds
type ListUploadBuildsParams struct {
	UploadID int64

	// Optional
	Credentials GameCredentials
}

// ListUploadBuildsResponse : response for ListUploadBuilds
type ListUploadBuildsResponse struct {
	Builds []*Build `json:"builds"`
}

// ListUploadBuilds lists recent builds for a given upload, by ID.
func (c *Client) ListUploadBuilds(params ListUploadBuildsParams) (*ListUploadBuildsResponse, error) {
	q := NewQuery(c, "/uploads/%d/builds", params.UploadID)
	q.AddGameCredentials(params.Credentials)
	r := &ListUploadBuildsResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

// GetBuildParams : params for GetBuild
type GetBuildParams struct {
	BuildID int64

	// Optional
	Credentials GameCredentials
}

// GetBuildResponse : response for GetBuild
type GetBuildResponse struct {
	Build *Build `json:"build"`
}

// GetBuild retrieves info about a single build, by ID.
func (c *Client) GetBuild(p GetBuildParams) (*GetBuildResponse, error) {
	q := NewQuery(c, "/builds/%d", p.BuildID)
	q.AddGameCredentials(p.Credentials)
	r := &GetBuildResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

// GetBuildUpgradePathParams : params for GetBuildUpgradePath
type GetBuildUpgradePathParams struct {
	CurrentBuildID int64
	TargetBuildID  int64

	// Optional
	Credentials GameCredentials
}

// GetBuildUpgradePathResponse : response for GetBuildUpgradePath
type GetBuildUpgradePathResponse struct {
	UpgradePath *UpgradePath `json:"upgradePath"`
}

// UpgradePath is a series of builds for which a (n,n+1) patch exists,
type UpgradePath struct {
	Builds []*Build `json:"builds"`
}

// GetBuildUpgradePath returns the complete list of builds one
// needs to go through to go from one version to another.
// It only works when upgrading (at the time of this writing).
func (c *Client) GetBuildUpgradePath(p GetBuildUpgradePathParams) (*GetBuildUpgradePathResponse, error) {
	q := NewQuery(c, "/builds/%d/upgrade-paths/%d", p.CurrentBuildID, p.TargetBuildID)
	q.AddGameCredentials(p.Credentials)
	r := &GetBuildUpgradePathResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

// NewDownloadSessionParams : params for NewDownloadSession
type NewDownloadSessionParams struct {
	GameID int64

	Credentials GameCredentials
}

// NewDownloadSessionResponse : response for NewDownloadSession
type NewDownloadSessionResponse struct {
	UUID string `json:"uuid"`
}

// NewDownloadSession creates a new download session. It is used
// for more accurate download analytics. Downloading multiple patch
// and signature files may all be part of the same "download session":
// upgrading a game to its latest version. It should only count as one download.
func (c *Client) NewDownloadSession(p NewDownloadSessionParams) (*NewDownloadSessionResponse, error) {
	q := NewQuery(c, "/games/%d/download-sessions", p.GameID)
	q.AddGameCredentials(p.Credentials)
	r := &NewDownloadSessionResponse{}
	return r, q.Post(r)
}

//-------------------------------------------------------

// MakeUploadDownloadURLParams : params for MakeUploadDownloadURL
type MakeUploadDownloadURLParams struct {
	UploadID int64

	// Optional
	UUID string

	// Optional
	Credentials GameCredentials
}

// MakeUploadDownloadURL generates a download URL for an upload
func (c *Client) MakeUploadDownloadURL(p MakeUploadDownloadURLParams) string {
	q := NewQuery(c, "uploads/%d/download", p.UploadID)
	q.AddAPICredentials()
	q.AddGameCredentials(p.Credentials)
	q.AddStringIfNonEmpty("uuid", p.UUID)
	return q.URL()
}

//-------------------------------------------------------

// MakeBuildDownloadURLParams : params for MakeBuildDownloadURL
type MakeBuildDownloadURLParams struct {
	BuildID int64
	Type    BuildFileType

	// Optional: Defaults to BuildFileSubTypeDefault
	SubType BuildFileSubType

	// Optional
	UUID string

	// Optional
	Credentials GameCredentials
}

// MakeBuildDownloadURL generates as download URL for a specific build
func (c *Client) MakeBuildDownloadURL(p MakeBuildDownloadURLParams) string {
	subType := p.SubType
	if subType == "" {
		subType = BuildFileSubTypeDefault
	}

	q := NewQuery(c, "builds/%d/download/%s/%s", p.BuildID, p.Type, subType)
	q.AddAPICredentials()
	q.AddGameCredentials(p.Credentials)
	q.AddStringIfNonEmpty("uuid", p.UUID)
	return q.URL()
}
