package models

type ProfileData struct {
	ProfileID int64 `json:"profileId" hades:"primary_key"`

	Key   string `json:"string" hades:"primary_key"`
	Value string `json:"value"`
}
