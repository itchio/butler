package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/mitchellh/mapstructure"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/manager"
	itchio "github.com/itchio/go-itchio"
)

var tr *JSONTransport

func cave() {
	must(doCave())
}

func doCave() error {
	tr = NewJSONTransport()
	tr.Start()

	var command CaveCommand
	err := readMessage("cave-command", &command)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	switch command.Operation {
	case CaveCommandOperationInstall:
		return doCaveInstall(command.InstallParams)
	default:
		return fmt.Errorf("Unknown cave command operation '%s'", command.Operation)
	}
}

func readMessage(msgType string, res interface{}) error {
	msg, err := tr.Read(msgType)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  res,
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = decoder.Decode(msg)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	return nil
}

// CaveCommandOperation describes the operation butler should do
// for a specific cave: install it, launch it, etc.
type CaveCommandOperation string

const (
	// CaveCommandOperationInstall describes a download+install operation
	CaveCommandOperationInstall CaveCommandOperation = "install"
)

// CaveCommand describes a cave-related command butler should perform
type CaveCommand struct {
	Operation     CaveCommandOperation `json:"operation"`
	InstallParams *CaveInstallParams   `json:"installParams"`
}

// CaveInstallParams contains all the parameters needed to perform
// an installation for a game
type CaveInstallParams struct {
	Game          *itchio.Game     `json:"game"`
	StageFolder   string           `json:"stageFolder"`
	InstallFolder string           `json:"installFolder"`
	Upload        *itchio.Upload   `json:"upload"`
	Build         *itchio.Build    `json:"build"`
	Credentials   *CaveCredentials `json:"credentials"`
}

// CaveCredentials contains all the credentials required to make API requests
// including the download key if any
type CaveCredentials struct {
	Server      string `json:"server"`
	APIKey      string `json:"apiKey"`
	DownloadKey int64  `json:"downloadKey"`
}

func doCaveInstall(installParams *CaveInstallParams) error {
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
	err = doCp(archiveUrl, archiveDownloadPath, true)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return errors.New("stub")
}

func clientFromCredentials(credentials *CaveCredentials) (*itchio.Client, error) {
	if credentials == nil {
		return nil, errors.New("Missing credentials")
	}

	if credentials.APIKey == "" {
		return nil, errors.New("Missing API key in credentials")
	}

	client := itchio.ClientWithKey(credentials.APIKey)

	if credentials.Server != "" {
		client.SetServer(credentials.Server)
	}

	return client, nil
}

type CommandError struct {
	Type      string `json:"type"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Operation string `json:"operation"`
}

func (ce *CommandError) Error() string {
	return fmt.Sprintf("command %s error %s: %s", ce.Operation, ce.Code, ce.Message)
}

func (ce *CommandError) Throw() error {
	ce.Type = "command-error"
	comm.Result(ce)

	return errors.Wrap(ce, 1)
}

type PickUploadParams struct {
	Uploads []*itchio.Upload `json:"uploads"`
}

type PickUploadResult struct {
	Index int64
}
