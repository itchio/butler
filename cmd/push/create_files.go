package push

import (
	"fmt"
	"net/http"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/go-itchio"
)

type fileSlot struct {
	Type     itchio.BuildFileType
	Response *itchio.CreateBuildFileResponse
}

func createBothFiles(client *itchio.Client, buildID int64) (patch *itchio.CreateBuildFileResponse, signature *itchio.CreateBuildFileResponse, err error) {
	createFile := func(buildType itchio.BuildFileType, done chan fileSlot, errs chan error) {
		res, err := client.CreateBuildFile(buildID, buildType, itchio.BuildFileSubTypeDefault, itchio.FileUploadTypeDeferredResumable)
		if err != nil {
			errs <- errors.Wrap(err, 1)
		}
		comm.Debugf("Created %s build file: %+v", buildType, res.File)

		// TODO: resumable upload session creation sounds like it belongs in an external lib, go-itchio maybe?
		req, reqErr := http.NewRequest("POST", res.File.UploadURL, nil)
		if reqErr != nil {
			errs <- errors.Wrap(reqErr, 1)
		}

		req.ContentLength = 0

		for k, v := range res.File.UploadHeaders {
			req.Header.Add(k, v)
		}

		gcsRes, gcsErr := client.HTTPClient.Do(req)
		if gcsErr != nil {
			errs <- errors.Wrap(gcsErr, 1)
		}

		if gcsRes.StatusCode != 201 {
			errs <- errors.Wrap(fmt.Errorf("could not create resumable upload session (got HTTP %d)", gcsRes.StatusCode), 1)
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
			err = errors.Wrap(err, 1)
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
