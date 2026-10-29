package operate

import (
	"github.com/itchio/butler/butlerd"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/headway/state"
)

func MakeSourceURL(client *itchio.Client, consumer *state.Consumer, sessionID string, params *InstallParams, fileType string) string {
	return makeSourceURL(client, sessionID, params, fileType, false)
}

func MakeProbeSourceURL(client *itchio.Client, consumer *state.Consumer, sessionID string, params *InstallParams, fileType string) string {
	return makeSourceURL(client, sessionID, params, fileType, true)
}

func makeSourceURL(client *itchio.Client, sessionID string, params *InstallParams, fileType string, probe bool) string {
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
			Probe:       probe,
		})
	}

	return client.MakeUploadDownloadURL(itchio.MakeUploadDownloadURLParams{
		UploadID:    params.Upload.ID,
		UUID:        sessionID,
		Credentials: params.Access.Credentials,
		Probe:       probe,
	})
}

type ItchfsURLParams struct {
	Credentials *butlerd.GameCredentials
	UploadID    int64
	BuildID     int64
	FileType    string
	UUID        string
}
