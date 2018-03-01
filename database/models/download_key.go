package models

import itchio "github.com/itchio/go-itchio"

type DownloadKey struct {
	ID int64 `json:"id"`

	GameID int64        `json:"gameId"`
	Game   *itchio.Game `json:"game,omitempty"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`

	OwnerID int64        `json:"ownerId"`
	Owner   *itchio.User `json:"owner,omitempty"`
}
