package models

import itchio "github.com/itchio/go-itchio"

type GameUpload struct {
	GameID int64        `json:"gameId" hades:"primary_key"`
	Game   *itchio.Game `json:"game"`

	UploadID int64          `json:"uploadId" hades:"primary_key"`
	Upload   *itchio.Upload `json:"upload"`
}
