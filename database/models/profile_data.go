package models

type ProfileData struct {
	ProfileID int64  `json:"profileId" gorm:"primary_key;auto_increment:false"`
	Key       string `json:"string" gorm:"primary_key"`
	Value     string `json:"value"`
}
