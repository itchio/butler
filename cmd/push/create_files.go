package push

import (
	"net/http"

	"github.com/itchio/butler/comm"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

type createBothFilesResponse struct {
	patchRes     *itchio.CreateBuildFileResponse
	signatureRes *itchio.CreateBuildFileResponse
}

func createBothFiles(client *itchio.Client, buildID int64) (*createBothFilesResponse, error) {
	createFile := func(buildType itchio.BuildFileType, result **itchio.CreateBuildFileResponse) error {
		buildFileRes, err := client.CreateBuildFile(itchio.CreateBuildFileParams{
			BuildID:        buildID,
			Type:           buildType,
			SubType:        itchio.BuildFileSubTypeDefault,
			FileUploadType: itchio.FileUploadTypeDeferredResumable,
		})
		if err != nil {
			return errors.WithMessage(err, "creating build file on server")
		}
		comm.Debugf("Created %s build file: %+v", buildType, buildFileRes.File)

		// TODO: resumable upload session creation sounds like it belongs in an external lib, go-itchio maybe?
		req, err := http.NewRequest("POST", buildFileRes.File.UploadURL, nil)
		if err != nil {
			return errors.WithMessage(err, "getting resumable upload session parameters")
		}

		req.ContentLength = 0

		for k, v := range buildFileRes.File.UploadHeaders {
			req.Header.Add(k, v)
		}

		gcsRes, err := client.HTTPClient.Do(req)
		if err != nil {
			return errors.WithMessage(err, "creating resumable upload session")
		}

		if gcsRes.StatusCode != 201 {
			return errors.Errorf("while creating resumable upload session, got HTTP %d", gcsRes.StatusCode)
		}

		buildFileRes.File.UploadHeaders = nil
		buildFileRes.File.UploadURL = gcsRes.Header.Get("Location")
		comm.Debugf("Started resumable upload session %s", buildFileRes.File.UploadURL)

		*result = buildFileRes
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
