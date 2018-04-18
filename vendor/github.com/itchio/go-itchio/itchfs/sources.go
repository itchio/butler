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

func serveBuildFile(r *itchio.DownloadUploadBuildResponse, fileType string) (string, error) {
	switch fileType {
	case "archive":
		return r.Archive.URL, nil
	case "patch":
		return r.Patch.URL, nil
	case "signature":
		return r.Signature.URL, nil
	case "manifest":
		return r.Manifest.URL, nil
	case "unpacked":
		return r.Unpacked.URL, nil
	}

	return "", fmt.Errorf("unknown file type %s", fileType)
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

func stripDownloadKey(in url.Values) url.Values {
	res := url.Values{}
	for k, vv := range in {
		if k == "download_key_id" {
			continue
		}
		for _, v := range vv {
			res.Add(k, v)
		}
	}
	return res
}

func (s *source) makeDownloadBuildURL(tokens []string) (htfs.GetURLFunc, error) {
	uploadID, err := strconv.ParseInt(tokens[2], 10, 64)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	buildID, err := strconv.ParseInt(tokens[5], 10, 64)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	downloadKey := s.QueryValues.Get("download_key_id")

	fileType := tokens[6]

	getter := func() (string, error) {
		r, err := s.ItchClient.DownloadUploadBuildWithKeyAndValues(downloadKey, uploadID, buildID, stripDownloadKey(s.QueryValues))
		if err != nil {
			return "", err
		}

		return serveBuildFile(r, fileType)
	}
	return getter, nil
}

func (s *source) makeKeyDownloadBuildURL(tokens []string) (htfs.GetURLFunc, error) {
	downloadKey := tokens[2]

	uploadID, err := strconv.ParseInt(tokens[4], 10, 64)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	buildID, err := strconv.ParseInt(tokens[6], 10, 64)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	fileType := tokens[7]

	getter := func() (string, error) {
		r, err := s.ItchClient.DownloadUploadBuildWithKeyAndValues(downloadKey, uploadID, buildID, stripDownloadKey(s.QueryValues))
		if err != nil {
			return "", err
		}

		return serveBuildFile(r, fileType)
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
		r, err := s.ItchClient.GetBuildFileDownloadURLWithValues(buildID, buildFileID, stripDownloadKey(s.QueryValues))
		if err != nil {
			return "", err
		}

		return r.URL, nil
	}
	return getter, nil
}

func (s *source) makeDownloadUploadURL(tokens []string) (htfs.GetURLFunc, error) {
	uploadID, err := strconv.ParseInt(tokens[2], 10, 64)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	downloadKey := s.QueryValues.Get("download_key_id")

	getter := func() (string, error) {
		r, err := s.ItchClient.UploadDownloadWithKeyAndValues(downloadKey, uploadID, stripDownloadKey(s.QueryValues))
		if err != nil {
			return "", err
		}

		return r.URL, nil
	}
	return getter, nil
}

func (s *source) makeKeyDownloadUploadURL(tokens []string) (htfs.GetURLFunc, error) {
	downloadKey := tokens[2]

	uploadID, err := strconv.ParseInt(tokens[4], 10, 64)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	getter := func() (string, error) {
		r, err := s.ItchClient.UploadDownloadWithKeyAndValues(downloadKey, uploadID, stripDownloadKey(s.QueryValues))
		if err != nil {
			return "", err
		}

		return r.URL, nil
	}
	return getter, nil
}
