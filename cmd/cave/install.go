package cave

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/cp"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/manager"
	"github.com/itchio/butler/mansion"
	itchio "github.com/itchio/go-itchio"
)

func doCaveInstall(ctx *mansion.Context, installParams *CaveInstallParams) error {
	if installParams == nil {
		return errors.New("Missing install params")
	}

	if installParams.Game == nil {
		return errors.New("Missing game in install")
	}

	if installParams.InstallFolder == "" {
		return errors.New("Missing install folder in install")
	}

	if installParams.StageFolder == "" {
		return errors.New("Missing stage folder in install")
	}

	comm.Opf("Installing game %s", installParams.Game.Title)
	comm.Logf("into directory %s", installParams.InstallFolder)
	comm.Logf("using stage directory %s", installParams.StageFolder)

	err := os.MkdirAll(installParams.StageFolder, 0755)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	client, err := clientFromCredentials(installParams.Credentials)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if installParams.Upload == nil {
		comm.Logf("No upload specified, looking for compatible ones...")
		uploads, err := client.ListGameUploads(&itchio.ListGameUploadsParams{
			GameID:        installParams.Game.ID,
			DownloadKeyID: installParams.Credentials.DownloadKey,
		})
		if err != nil {
			return errors.Wrap(err, 0)
		}

		comm.Logf("Got %d uploads, here they are:", len(uploads.Uploads))
		for _, upload := range uploads.Uploads {
			comm.Logf("- %#v", upload)
		}

		uploadsFilterResult := manager.NarrowDownUploads(uploads.Uploads, installParams.Game, manager.CurrentRuntime())
		comm.Logf("After filter, got %d uploads, they are: ", len(uploadsFilterResult.Uploads))
		for _, upload := range uploadsFilterResult.Uploads {
			comm.Logf("- %#v", upload)
		}

		if len(uploadsFilterResult.Uploads) == 0 {
			return (&CommandError{
				Code:      "noCompatibleUploads",
				Message:   "No compatible uploads",
				Operation: "install",
			}).Throw()
		}

		if len(uploadsFilterResult.Uploads) == 1 {
			installParams.Upload = uploadsFilterResult.Uploads[0]
		} else {
			comm.Request("install", "pick-upload", &PickUploadParams{
				Uploads: uploadsFilterResult.Uploads,
			})

			var r PickUploadResult
			err := readMessage("pick-upload-result", &r)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			installParams.Upload = uploadsFilterResult.Uploads[r.Index]
		}
	}

	var archiveUrlPath string
	if installParams.Build == nil {
		archiveUrlPath = fmt.Sprintf("/upload/%d/download", installParams.Upload.ID)
	} else {
		archiveUrlPath = fmt.Sprintf("/upload/%d/download/builds/%d/archive", installParams.Upload.ID, installParams.Build.ID)
	}
	values := make(url.Values)
	values.Set("api_key", installParams.Credentials.APIKey)
	if installParams.Credentials.DownloadKey != 0 {
		values.Set("download_key", fmt.Sprintf("%d", installParams.Credentials.DownloadKey))
	}
	var archiveUrl = fmt.Sprintf("itchfs://%s?%s", archiveUrlPath, values.Encode())

	// use natural file name for non-wharf downloads
	var archiveDownloadName = installParams.Upload.Filename
	if installParams.Build != nil {
		// make up a sensible .zip name for wharf downloads
		archiveDownloadName = fmt.Sprintf("%d-%d.zip", installParams.Upload.ID, installParams.Build.ID)
	}

	var archiveDownloadPath = filepath.Join(installParams.StageFolder, archiveDownloadName)
	err = cp.Do(ctx, archiveUrl, archiveDownloadPath, true)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	installerType, err := getInstallerType(archiveDownloadPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Logf("Will use installer %s", installerType)

	return errors.New("stub")
}
