package models

type DashboardGame struct {
	GameID int64 `json:"gameId"`
	UserID int64 `json:"userId"`

	Order int64 `json:"order"`

	// Stats

	ViewsCount     int64 `json:"viewsCount"`
	DownloadsCount int64 `json:"downloadsCount"`
	PurchasesCount int64 `json:"purchasesCount"`

	Published bool `json:"published"`
}
