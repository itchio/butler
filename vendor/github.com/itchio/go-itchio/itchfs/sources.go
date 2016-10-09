package itchfs

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/go-errors/errors"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/httpkit/httpfile"
)

// over-engineering follows

type SourceType int

const (
	// 691 is the earliest Bernoulli number that's also an irregular prime.
	// It's also sufficiently different from zero that it should surface
	// coding errors more easily
	SourceType_DownloadBuild SourceType = 691 + iota
	SourceType_KeyDownloadBuild
	SourceType_WharfDownloadBuild
	SourceType_DownloadUpload
)

type Source struct {
	Type       SourceType
	ItchClient *itchio.Client
	Path       string
}

var patterns = map[string]SourceType{
	"/upload/*/download":                    SourceType_DownloadUpload,
	"/upload/*/download/builds/*/*":         SourceType_DownloadBuild,
	"/download-key/*/download/*/builds/*/*": SourceType_KeyDownloadBuild,
	"/wharf/builds/*/files/*/download":      SourceType_WharfDownloadBuild,
}

func ObtainSource(itchClient *itchio.Client, itchPath string) (*Source, error) {
	var matches bool
	var err error

	for pattern, sourceType := range patterns {
		matches, err = path.Match(pattern, itchPath)
		if err != nil {
			return nil, err
		}

		if matches {
			return &Source{
				Type:       sourceType,
				ItchClient: itchClient,
				Path:       itchPath,
			}, nil
		}
	}

	return nil, fmt.Errorf("unrecognized itchfs pattern: %s", itchPath)
}

func serveBuildFile(r itchio.DownloadUploadBuildResponse, fileType string) (string, error) {
	switch fileType {
	case "archive":
		return r.Archive.URL, nil
	case "patch":
		return r.Patch.URL, nil
	case "signature":
		return r.Signature.URL, nil
	case "manifest":
		return r.Manifest.URL, nil
	}

	return "", fmt.Errorf("unknown file type %s", fileType)
}

func (s *Source) makeGetURL() (httpfile.GetURLFunc, error) {
	tokens := strings.Split(s.Path, "/")

	var getter httpfile.GetURLFunc

	switch s.Type {
	case SourceType_DownloadBuild:
		uploadID, err := strconv.ParseInt(tokens[2], 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		buildID, err := strconv.ParseInt(tokens[5], 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		fileType := tokens[6]

		getter = func() (string, error) {
			r, err := s.ItchClient.DownloadUploadBuild(uploadID, buildID)
			if err != nil {
				return "", err
			}

			return serveBuildFile(r, fileType)
		}
	case SourceType_KeyDownloadBuild:
		downloadKey := tokens[2]

		uploadID, err := strconv.ParseInt(tokens[4], 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		buildID, err := strconv.ParseInt(tokens[6], 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		fileType := tokens[7]

		getter = func() (string, error) {
			r, err := s.ItchClient.DownloadUploadBuildWithKey(downloadKey, uploadID, buildID)
			if err != nil {
				return "", err
			}

			return serveBuildFile(r, fileType)
		}
	case SourceType_WharfDownloadBuild:
		buildID, err := strconv.ParseInt(tokens[3], 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		buildFileID, err := strconv.ParseInt(tokens[5], 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		getter = func() (string, error) {
			r, err := s.ItchClient.GetBuildFileDownloadURL(buildID, buildFileID)
			if err != nil {
				return "", err
			}

			return r.URL, nil
		}
	case SourceType_DownloadUpload:
		uploadID, err := strconv.ParseInt(tokens[2], 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		getter = func() (string, error) {
			r, err := s.ItchClient.UploadDownload(uploadID)
			if err != nil {
				return "", err
			}

			return r.URL, nil
		}
	default:
		return nil, fmt.Errorf("unsupported source type: %d", s.Type)
	}

	return getter, nil
}
