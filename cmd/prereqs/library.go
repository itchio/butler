package prereqs

import (
	"fmt"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/operate"
	itchio "github.com/itchio/go-itchio"
)

// https://fasterthanlime.itch.io/itch-redists
const RedistsGameID int64 = 222417

type Library interface {
	GetURL(name string, fileType string) (string, error)
}

type library struct {
	credentials *buse.GameCredentials
	uploads     []*itchio.Upload
}

var _ Library = (*library)(nil)

func NewLibrary(credentials *buse.GameCredentials) (Library, error) {
	client, err := operate.ClientFromCredentials(credentials)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	uploadsRes, err := client.ListGameUploads(&itchio.ListGameUploadsParams{
		GameID: RedistsGameID,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	l := &library{
		credentials: credentials,
		uploads:     uploadsRes.Uploads,
	}

	return l, nil
}

func (l *library) GetURL(name string, fileType string) (string, error) {
	upload := l.getUpload(name)
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

func (l *library) getUpload(name string) *itchio.Upload {
	for _, upload := range l.uploads {
		if upload.ChannelName == name {
			return upload
		}
	}

	return nil
}
