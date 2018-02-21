package models

type DownloadKey struct {
	ID int64 `json:"id"`

	GameID    int64  `json:"gameId"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`

	OwnerID int64 `json:"ownerId"`
}
