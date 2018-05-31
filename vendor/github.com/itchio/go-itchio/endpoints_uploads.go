package itchio

type GameCredentials struct {
	DownloadKeyID int64  `json:"download_key_id,omitempty"`
	Password      string `json:"password,omitempty"`
	Secret        string `json:"secret,omitempty"`
}

//-------------------------------------------------------

type ListGameUploadsParams struct {
	GameID int64

	// Optional
	Credentials GameCredentials
}

// ListGameUploadsResponse is what the server replies with when asked for a game's uploads
type ListGameUploadsResponse struct {
	Uploads []*Upload `json:"uploads"`
}

// ListGameUploads lists the uploads for a game that we have access to with our API key
func (c *Client) ListGameUploads(p *ListGameUploadsParams) (*ListGameUploadsResponse, error) {
	q := NewQuery(c, "/games/%d/uploads", p.GameID)
	q.AddGameCredentials(p.Credentials)
	r := &ListGameUploadsResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

type GetUploadParams struct {
	UploadID int64

	// Optional
	Credentials GameCredentials
}

type GetUploadResponse struct {
	Upload *Upload `json:"upload"`
}

func (c *Client) GetUpload(params *GetUploadParams) (*GetUploadResponse, error) {
	q := NewQuery(c, "/uploads/%d", params.UploadID)
	q.AddGameCredentials(params.Credentials)
	r := &GetUploadResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

type ListUploadBuildsParams struct {
	UploadID int64

	// Optional
	Credentials GameCredentials
}

type ListUploadBuildsResponse struct {
	Builds []*Build `json:"builds"`
}

func (c *Client) ListUploadBuilds(params *ListUploadBuildsParams) (*ListUploadBuildsResponse, error) {
	q := NewQuery(c, "/uploads/%d/builds", params.UploadID)
	q.AddGameCredentials(params.Credentials)
	r := &ListUploadBuildsResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

type GetBuildParams struct {
	BuildID int64

	// Optional
	Credentials GameCredentials
}

type GetBuildResponse struct {
	Build *Build `json:"build"`
}

func (c *Client) GetBuild(p *GetBuildParams) (*GetBuildResponse, error) {
	q := NewQuery(c, "/builds/%d", p.BuildID)
	q.AddGameCredentials(p.Credentials)
	r := &GetBuildResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

type GetBuildUpgradePathParams struct {
	CurrentBuildID int64
	TargetBuildID  int64

	// Optional
	Credentials GameCredentials
}

type GetBuildUpgradePathResponse struct {
	UpgradePath *UpgradePath `json:"upgradePath"`
}

type UpgradePath struct {
	Builds []*Build `json:"builds"`
}

func (c *Client) GetBuildUpgradePath(p *GetBuildUpgradePathParams) (*GetBuildUpgradePathResponse, error) {
	q := NewQuery(c, "/builds/%d/upgrade-paths/%d", p.CurrentBuildID, p.TargetBuildID)
	q.AddGameCredentials(p.Credentials)
	r := &GetBuildUpgradePathResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

type NewDownloadSessionParams struct {
	GameID int64

	Credentials GameCredentials
}

func (c *Client) NewDownloadSession(p *NewDownloadSessionParams) (*NewDownloadSessionResponse, error) {
	q := NewQuery(c, "/games/%d/download-sessions", p.GameID)
	q.AddGameCredentials(p.Credentials)
	r := &NewDownloadSessionResponse{}
	return r, q.Post(r)
}

//-------------------------------------------------------

type MakeUploadDownloadParams struct {
	UploadID int64

	// Optional
	UUID string

	// Optional
	Credentials GameCredentials
}

func (c *Client) MakeUploadDownloadURL(p *MakeUploadDownloadParams) string {
	q := NewQuery(c, "uploads/%d/download", p.UploadID)
	q.AddAPICredentials()
	q.AddGameCredentials(p.Credentials)
	q.AddStringIfNonEmpty("uuid", p.UUID)
	return q.URL()
}

//-------------------------------------------------------

type MakeBuildDownloadParams struct {
	BuildID int64
	Type    BuildFileType

	// Optional: Defaults to BuildFileSubTypeDefault
	SubType BuildFileSubType

	// Optional
	UUID string

	// Optional
	Credentials GameCredentials
}

func (c *Client) MakeBuildDownloadURL(p *MakeBuildDownloadParams) string {
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
