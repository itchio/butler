package prereqs

import (
	"fmt"

	"github.com/itchio/butler/butlerd"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/ox"
	"github.com/pkg/errors"
)

type Library interface {
	GetURL(name string, fileType itchio.BuildFileType) (string, error)
	GetUpload(name string) *itchio.Upload
}

type library struct {
	client  *itchio.Client
	uploads []*itchio.Upload
	runtime ox.Runtime
}

var _ Library = (*library)(nil)

func NewLibrary(rc *butlerd.RequestContext, runtime ox.Runtime, apiKey string) (Library, error) {
	client := rc.Client(apiKey)

	uploadsRes, err := client.ListGameUploads(rc.Ctx, itchio.ListGameUploadsParams{
		GameID: RedistsGame.ID,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	l := &library{
		client:  client,
		uploads: uploadsRes.Uploads,
		runtime: runtime,
	}

	return l, nil
}

func (l *library) GetURL(name string, fileType itchio.BuildFileType) (string, error) {
	upload := l.GetUpload(name)
	if upload == nil {
		return "", fmt.Errorf("Could not find download for prereq (%s)", name)
	}

	url := l.client.MakeBuildDownloadURL(itchio.MakeBuildDownloadURLParams{
		BuildID: upload.Build.ID,
		Type:    fileType,
	})
	return url, nil
}

func (l *library) GetUpload(name string) *itchio.Upload {
	channelName := fmt.Sprintf("%s-%s", name, l.runtime.Platform)

	for _, upload := range l.uploads {
		if upload.ChannelName == channelName {
			return upload
		}
	}

	return nil
}
