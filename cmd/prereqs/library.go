package prereqs

import (
	"fmt"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

type Library interface {
	GetURL(name string, fileType string) (string, error)
	GetUpload(name string) *itchio.Upload
	GetCredentials() *butlerd.GameCredentials
}

type library struct {
	credentials *butlerd.GameCredentials
	uploads     []*itchio.Upload
}

var _ Library = (*library)(nil)

func NewLibrary(rc *butlerd.RequestContext, credentials *butlerd.GameCredentials) (Library, error) {
	client := rc.ClientFromCredentials(credentials)

	uploadsRes, err := client.ListGameUploads(&itchio.ListGameUploadsParams{
		GameID: RedistsGame.ID,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	l := &library{
		credentials: credentials,
		uploads:     uploadsRes.Uploads,
	}

	return l, nil
}

func (l *library) GetURL(name string, fileType string) (string, error) {
	upload := l.GetUpload(name)
	if upload == nil {
		return "", fmt.Errorf("Could not find download for prereq (%s)", name)
	}

	url := operate.MakeItchfsURL(&operate.ItchfsURLParams{
		Credentials: l.credentials,
		UploadID:    upload.ID,
		BuildID:     upload.Build.ID,
		FileType:    fileType,
	})
	return url, nil
}

func (l *library) GetUpload(name string) *itchio.Upload {
	for _, upload := range l.uploads {
		if upload.ChannelName == name {
			return upload
		}
	}

	return nil
}

func (l *library) GetCredentials() *butlerd.GameCredentials {
	return l.credentials
}
