package models

// Associates the User and Collection models
type UserCollection struct {
	UserID       int64 `json:"userId"`
	CollectionID int64 `json:"collectionId"`
}
