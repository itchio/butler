package itchfs

import (
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/httpkit/htfs"
	"github.com/pkg/errors"
)

// over-engineering follows

type sourceType int

const (
	// 691 is the earliest Bernoulli number that's also an irregular prime.
	// It's also sufficiently different from zero that it should surface
	// coding errors more easily
	sourceTypeDownloadBuild sourceType = 691 + iota
	sourceTypeKeyDownloadBuild
	sourceTypeWharfDownloadBuild
	sourceTypeDownloadUpload
	sourceTypeKeyDownloadUpload
)

type source struct {
	Type        sourceType
	ItchClient  *itchio.Client
	Path        string
	QueryValues url.Values
}

var patterns = map[string]sourceType{
	"/upload/*/download":            sourceTypeDownloadUpload, // legacy
	"/upload/*/download/builds/*/*": sourceTypeDownloadBuild,  // legacy

	"/uploads/*/download":            sourceTypeDownloadUpload,
	"/uploads/*/download/builds/*/*": sourceTypeDownloadBuild,

	"/download-key/*/download/*":            sourceTypeKeyDownloadUpload, // deprecated
	"/download-key/*/download/*/builds/*/*": sourceTypeKeyDownloadBuild,  // deprecated

	"/wharf/builds/*/files/*/download": sourceTypeWharfDownloadBuild,
}

func obtainSource(itchClient *itchio.Client, itchPath string, queryValues url.Values) (*source, error) {
	var matches bool
	var err error

	for pattern, sourceType := range patterns {
		matches, err = path.Match(pattern, itchPath)
		if err != nil {
			return nil, err
		}

		if matches {
			return &source{
				Type:        sourceType,
				ItchClient:  itchClient,
				Path:        itchPath,
				QueryValues: queryValues,
			}, nil
		}
	}

	return nil, fmt.Errorf("unrecognized itchfs pattern: %s", itchPath)
}

func (s *source) makeGetURL() (htfs.GetURLFunc, error) {
	tokens := strings.Split(s.Path, "/")

	switch s.Type {
	case sourceTypeDownloadBuild:
		return s.makeDownloadBuildURL(tokens)
	// deprecated
	case sourceTypeKeyDownloadBuild:
		return s.makeKeyDownloadBuildURL(tokens)
	case sourceTypeWharfDownloadBuild:
		return s.makeWharfDownloadBuildURL(tokens)
	case sourceTypeDownloadUpload:
		return s.makeDownloadUploadURL(tokens)
	// deprecated
	case sourceTypeKeyDownloadUpload:
		return s.makeKeyDownloadUploadURL(tokens)
	default:
		return nil, fmt.Errorf("unsupported source type: %d", s.Type)
	}
}

func (s *source) makeDownloadBuildURL(tokens []string) (htfs.GetURLFunc, error) {
	buildID, _ := strconv.ParseInt(tokens[5], 10, 64)
	fileType := tokens[6]

	getter := func() (string, error) {
		return s.ItchClient.MakeBuildDownloadURL(itchio.MakeBuildDownloadParams{
			BuildID:     buildID,
			UUID:        s.QueryValues.Get("uuid"),
			Type:        itchio.BuildFileType(fileType),
			Credentials: parseGameCredentials(s.QueryValues),
		}), nil
	}
	return getter, nil
}

func parseGameCredentials(values url.Values) itchio.GameCredentials {
	var creds itchio.GameCredentials

	for k, vv := range values {
		for _, v := range vv {
			if k == "download_key_id" {
				creds.DownloadKeyID, _ = strconv.ParseInt(v, 10, 64)
			}
		}
	}

	return creds
}

func (s *source) makeKeyDownloadBuildURL(tokens []string) (htfs.GetURLFunc, error) {
	downloadKey := tokens[2]
	buildID, _ := strconv.ParseInt(tokens[6], 10, 64)
	fileType := tokens[7]

	getter := func() (string, error) {
		creds := parseGameCredentials(s.QueryValues)
		creds.DownloadKeyID, _ = strconv.ParseInt(downloadKey, 10, 64)

		return s.ItchClient.MakeBuildDownloadURL(itchio.MakeBuildDownloadParams{
			BuildID:     buildID,
			Type:        itchio.BuildFileType(fileType),
			UUID:        s.QueryValues.Get("uuid"),
			Credentials: creds,
		}), nil
	}

	return getter, nil
}

func (s *source) makeWharfDownloadBuildURL(tokens []string) (htfs.GetURLFunc, error) {
	buildID, err := strconv.ParseInt(tokens[3], 10, 64)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	buildFileID, err := strconv.ParseInt(tokens[5], 10, 64)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	getter := func() (string, error) {
		return s.ItchClient.MakeBuildFileDownloadURL(itchio.MakeBuildFileDownloadURLParams{
			BuildID: buildID,
			FileID:  buildFileID,
		}), nil
	}
	return getter, nil
}

func (s *source) makeDownloadUploadURL(tokens []string) (htfs.GetURLFunc, error) {
	uploadID, _ := strconv.ParseInt(tokens[2], 10, 64)

	getter := func() (string, error) {
		return s.ItchClient.MakeUploadDownloadURL(itchio.MakeUploadDownloadParams{
			UploadID:    uploadID,
			Credentials: parseGameCredentials(s.QueryValues),
			UUID:        s.QueryValues.Get("uuid"),
		}), nil
	}
	return getter, nil
}

func (s *source) makeKeyDownloadUploadURL(tokens []string) (htfs.GetURLFunc, error) {
	downloadKey := tokens[2]
	uploadID, _ := strconv.ParseInt(tokens[4], 10, 64)

	creds := parseGameCredentials(s.QueryValues)
	creds.DownloadKeyID, _ = strconv.ParseInt(downloadKey, 10, 64)

	getter := func() (string, error) {
		return s.ItchClient.MakeUploadDownloadURL(itchio.MakeUploadDownloadParams{
			UploadID:    uploadID,
			Credentials: creds,
			UUID:        s.QueryValues.Get("uuid"),
		}), nil
	}
	return getter, nil
}
