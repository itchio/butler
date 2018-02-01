package operate

import (
	"fmt"
	"net/url"

	"github.com/itchio/butler/buse"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/wharf/state"
)

func sourceURL(consumer *state.Consumer, istate *InstallSubcontextState, params *buse.InstallParams, fileType string) string {
	var installSourceURLPath string
	if params.Build == nil {
		installSourceURLPath = fmt.Sprintf("/upload/%d/download", params.Upload.ID)
	} else {
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

		installSourceURLPath = fmt.Sprintf("/upload/%d/download/builds/%d/%s", params.Upload.ID, params.Build.ID, fileType)
	}
	values := make(url.Values)
	values.Set("api_key", params.Credentials.APIKey)
	values.Set("uuid", istate.DownloadSessionId)
	if params.Credentials.DownloadKey != 0 {
		values.Set("download_key_id", fmt.Sprintf("%d", params.Credentials.DownloadKey))
	}
	var installSourceURL = fmt.Sprintf("itchfs://%s?%s", installSourceURLPath, values.Encode())
	return installSourceURL
}
