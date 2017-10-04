package cave

import (
	"github.com/go-errors/errors"
	itchio "github.com/itchio/go-itchio"
	"github.com/mitchellh/mapstructure"
)

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

/////////////////////////////////////////////////
// Request/response pairs
/////////////////////////////////////////////////

type PickUploadParams struct {
	Uploads []*itchio.Upload `json:"uploads"`
}

type PickUploadResult struct {
	Index int64
}
