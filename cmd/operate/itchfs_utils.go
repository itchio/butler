package operate

import (
	"fmt"
	"net/url"

	"github.com/itchio/butler/buse"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/wharf/state"
)

func sourceURL(consumer *state.Consumer, istate *InstallSubcontextState, params *buse.InstallParams, fileType string) string {
	var buildID int64
	if params.Build != nil {
		buildID = params.Build.ID

		if fileType == "" {
			fileType = "archive"
			for _, bf := range params.Build.Files {
				if bf.Type == itchio.BuildFileTypeUnpacked && bf.State == itchio.BuildFileStateUploaded {
					consumer.Infof("Build %d / %d has an unpacked file", params.Upload.ID, params.Build.ID)
					fileType = "unpacked"
					break
				}
			}
		}
	}

	return MakeItchfsURL(&ItchfsURLParams{
		Credentials: params.Credentials,
		UploadID:    params.Upload.ID,
		BuildID:     buildID,
		FileType:    fileType,
		UUID:        istate.DownloadSessionId,
	})
}

type ItchfsURLParams struct {
	Credentials *buse.GameCredentials
	UploadID    int64
	BuildID     int64
	FileType    string
	UUID        string
}

func MakeItchfsURL(params *ItchfsURLParams) string {
	var path string
	if params.BuildID == 0 {
		path = fmt.Sprintf("/upload/%d/download", params.UploadID)
	} else {
		path = fmt.Sprintf("/upload/%d/download/builds/%d/%s", params.UploadID, params.BuildID, params.FileType)
	}

	values := make(url.Values)
	values.Set("api_key", params.Credentials.APIKey)
	if params.UUID != "" {
		values.Set("uuid", params.UUID)
	}
	if params.Credentials.DownloadKey != 0 {
		values.Set("download_key_id", fmt.Sprintf("%d", params.Credentials.DownloadKey))
	}
	return fmt.Sprintf("itchfs://%s?%s", path, values.Encode())
}
