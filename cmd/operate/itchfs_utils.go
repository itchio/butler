package operate

import (
	"github.com/itchio/butler/butlerd"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/wharf/state"
)

func sourceURL(client *itchio.Client, consumer *state.Consumer, istate *InstallSubcontextState, params *InstallParams, fileType string) string {
	build := params.Build
	if build != nil {
		if fileType == "" {
			fileType = "archive"
			if itchio.FindBuildFile(itchio.BuildFileTypeUnpacked, build.Files) != nil {
				fileType = "unpacked"
			}

			return client.MakeBuildDownloadURL(&itchio.MakeBuildDownloadParams{
				BuildID:     build.ID,
				UUID:        istate.DownloadSessionId,
				Credentials: params.Access.Credentials,
				Type:        itchio.BuildFileType(fileType),
			})
		}
	}

	return client.MakeUploadDownloadURL(&itchio.MakeUploadDownloadParams{
		UploadID:    params.Upload.ID,
		UUID:        istate.DownloadSessionId,
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
