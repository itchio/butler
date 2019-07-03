package operate

import (
	"github.com/itchio/butler/butlerd"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/headway/state"
)

func MakeSourceURL(client *itchio.Client, consumer *state.Consumer, sessionID string, params *InstallParams, fileType string) string {
	build := params.Build
	if build != nil {
		if fileType == "" {
			fileType = "archive"
		}

		return client.MakeBuildDownloadURL(itchio.MakeBuildDownloadURLParams{
			BuildID:     build.ID,
			UUID:        sessionID,
			Credentials: params.Access.Credentials,
			Type:        itchio.BuildFileType(fileType),
		})
	}

	return client.MakeUploadDownloadURL(itchio.MakeUploadDownloadURLParams{
		UploadID:    params.Upload.ID,
		UUID:        sessionID,
		Credentials: params.Access.Credentials,
	})
}

type ItchfsURLParams struct {
	Credentials *butlerd.GameCredentials
	UploadID    int64
	BuildID     int64
	FileType    string
	UUID        string
}
