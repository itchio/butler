package push

import (
	"net/http"

	"github.com/itchio/butler/comm"
	"github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

type fileSlot struct {
	Type     itchio.BuildFileType
	Response *itchio.CreateBuildFileResponse
}

func createBothFiles(client *itchio.Client, buildID int64) (patch *itchio.CreateBuildFileResponse, signature *itchio.CreateBuildFileResponse, err error) {
	createFile := func(buildType itchio.BuildFileType, done chan fileSlot, errs chan error) {
		res, err := client.CreateBuildFile(buildID, buildType, itchio.BuildFileSubTypeDefault, itchio.FileUploadTypeDeferredResumable)
		if err != nil {
			errs <- errors.Wrap(err, "creating build file on server")
		}
		comm.Debugf("Created %s build file: %+v", buildType, res.File)

		// TODO: resumable upload session creation sounds like it belongs in an external lib, go-itchio maybe?
		req, reqErr := http.NewRequest("POST", res.File.UploadURL, nil)
		if reqErr != nil {
			errs <- errors.Wrap(reqErr, "getting resumable upload session parameters")
		}

		req.ContentLength = 0

		for k, v := range res.File.UploadHeaders {
			req.Header.Add(k, v)
		}

		gcsRes, gcsErr := client.HTTPClient.Do(req)
		if gcsErr != nil {
			errs <- errors.Wrap(gcsErr, "creating resumable upload session")
		}

		if gcsRes.StatusCode != 201 {
			errs <- errors.Errorf("while creating resumable upload session, got HTTP %d", gcsRes.StatusCode)
		}

		comm.Debugf("Started resumable upload session %s", gcsRes.Header.Get("Location"))

		res.File.UploadHeaders = nil
		res.File.UploadURL = gcsRes.Header.Get("Location")

		done <- fileSlot{buildType, res}
	}

	done := make(chan fileSlot)
	errs := make(chan error)

	go createFile(itchio.BuildFileTypePatch, done, errs)
	go createFile(itchio.BuildFileTypeSignature, done, errs)

	for i := 0; i < 2; i++ {
		select {
		case err = <-errs:
			err = errors.WithStack(err)
			return
		case slot := <-done:
			switch slot.Type {
			case itchio.BuildFileTypePatch:
				patch = slot.Response
			case itchio.BuildFileTypeSignature:
				signature = slot.Response
			}
		}
	}

	return
}
