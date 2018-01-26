package itchio

// WharfStatusResponse is what the API responds with when we ask for
// the status of the wharf infrastructure
type WharfStatusResponse struct {
	Success bool `json:"success"`
}

// GetMeResponse is what the API server responds when we ask for the user's profile
type GetMeResponse struct {
	User *User `json:"user"`
}

// GameUploadsResponse is what the server replies with when asked for a game's uploads
type GameUploadsResponse struct {
	Uploads []*Upload `json:"uploads"`
}

// UploadDownloadResponse is what the API replies to when we ask to download an upload
type UploadDownloadResponse struct {
	URL string `json:"url"`
}

// ListMyGamesResponse is what the API server answers when we ask for what games
// an account develops.
type ListMyGamesResponse struct {
	Games []*Game `json:"games"`
}

// NewBuildResponse is what the API replies with when we create a new build
type NewBuildResponse struct {
	Build struct {
		ID          int64 `json:"id"`
		UploadID    int64 `json:"uploadId"`
		ParentBuild struct {
			ID int64 `json:"id"`
		} `json:"parentBuild"`
	}
}

// ListChannelsResponse is what the API responds with when we ask for all the
// channels of a particular game
type ListChannelsResponse struct {
	Channels map[string]*Channel `json:"channels"`
}

// GetChannelResponse is what the API responds with when we ask info about a channel
type GetChannelResponse struct {
	Channel *Channel `json:"channel"`
}

// ListBuildFilesResponse is what the API responds with when we ask for the files
// in a specific build
type ListBuildFilesResponse struct {
	Files []*BuildFile `json:"files"`
}

// CreateBuildFileResponse is what the API responds when we create a new build file
type CreateBuildFileResponse struct {
	File *FileUploadSpec `json:"file"`
}

// FileUploadSpec contains the info needed to upload one specific build file
type FileUploadSpec struct {
	ID            int64             `json:"id"`
	UploadURL     string            `json:"uploadUrl"`
	UploadParams  map[string]string `json:"uploadParams"`
	UploadHeaders map[string]string `json:"uploadHeaders"`
}

// FinalizeBuildFileResponse is what the API responds when we finalize a build file
type FinalizeBuildFileResponse struct{}

// DownloadBuildFileResponse is what the API responds with when we
// ask to download an upload
type DownloadBuildFileResponse struct {
	URL string `json:"url"`
}

// DownloadUploadBuildResponseItem contains download information for a specific
// build file
type DownloadUploadBuildResponseItem struct {
	URL string `json:"url"`
}

// DownloadUploadBuildResponse is what the API responds when we want to download
// a build
type DownloadUploadBuildResponse struct {
	// Patch is the download info for the wharf patch, if any
	Patch *DownloadUploadBuildResponseItem `json:"patch"`
	// Signature is the download info for the wharf signature, if any
	Signature *DownloadUploadBuildResponseItem `json:"signature"`
	// Manifest is reserved
	Manifest *DownloadUploadBuildResponseItem `json:"manifest"`
	// Archive is the download info for the .zip archive, if any
	Archive *DownloadUploadBuildResponseItem `json:"archive"`
	// Unpacked is the only file of the build, if it's a single file
	Unpacked *DownloadUploadBuildResponseItem `json:"unpacked"`
}

// CreateBuildEventResponse is what the API responds with when you create a new build event
type CreateBuildEventResponse struct {
}

// CreateBuildFailureResponse is what the API responds with when we mark a build as failed
type CreateBuildFailureResponse struct {
}

// ListBuildEventsResponse is what the API responds with when we ask for the list of events for a build
type ListBuildEventsResponse struct {
	Events []*BuildEvent `json:"events"`
}

type ListGameUploadsResponse struct {
	Uploads []*Upload `json:"uploads"`
}

type FindUpgradeResponse struct {
	// UpgradePath is a list of patches needed to upgrade to the latest version
	UpgradePath []*UpgradePathItem `json:"upgradePath"`
}

type UpgradePathItem struct {
	ID          int64  `json:"id"`
	UserVersion string `json:"userVersion"`
	UpdatedAt   string `json:"updatedAt"`
	PatchSize   int64  `json:"patchSize"`
}

type NewDownloadSessionResponse struct {
	UUID string `json:"uuid"`
}
