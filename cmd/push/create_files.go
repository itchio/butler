package push

import (
	"fmt"
	"net/http"

	"github.com/itchio/butler/comm"
	"github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

type createBothFilesResponse struct {
	patchRes     *itchio.CreateBuildFileResponse
	signatureRes *itchio.CreateBuildFileResponse
}

func createBothFiles(client *itchio.Client, buildID int64) (*createBothFilesResponse, error) {
	createFile := func(buildType itchio.BuildFileType, result **itchio.CreateBuildFileResponse) error {
		res, err := client.CreateBuildFile(&itchio.CreateBuildFileParams{
			BuildID:        buildID,
			Type:           buildType,
			SubType:        itchio.BuildFileSubTypeDefault,
			FileUploadType: itchio.FileUploadTypeDeferredResumable,
		})
		if err != nil {
			return errors.WithMessage(err, "creating build file on server")
		}
		comm.Debugf("Created %s build file: %+v", buildType, res.File)

		// TODO: resumable upload session creation sounds like it belongs in an external lib, go-itchio maybe?
		req, reqErr := http.NewRequest("POST", res.File.UploadURL, nil)
		if reqErr != nil {
			return errors.WithMessage(err, "getting resumable upload session parameters")
		}

		req.ContentLength = 0

		for k, v := range res.File.UploadHeaders {
			req.Header.Add(k, v)
		}

		gcsRes, gcsErr := client.HTTPClient.Do(req)
		if gcsErr != nil {
			return errors.WithMessage(err, "creating resumable upload session")
		}

		if gcsRes.StatusCode != 201 {
			return errors.WithMessage(err, fmt.Sprintf("while creating resumable upload session, got HTTP %d", gcsRes.StatusCode))
		}

		res.File.UploadHeaders = nil
		res.File.UploadURL = gcsRes.Header.Get("Location")
		comm.Debugf("Started resumable upload session %s", res.File.UploadURL)

		*result = res
		return nil
	}

	var res createBothFilesResponse

	done := make(chan error, 2)

	go func() { done <- createFile(itchio.BuildFileTypePatch, &res.patchRes) }()
	go func() { done <- createFile(itchio.BuildFileTypeSignature, &res.signatureRes) }()

	for i := 0; i < 2; i++ {
		err := <-done
		if err != nil {
			return nil, err
		}
	}
	return &res, nil
}
