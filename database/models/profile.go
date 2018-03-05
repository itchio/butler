package models

import (
	"time"

	itchio "github.com/itchio/go-itchio"
)

type Profile struct {
	ID int64 `json:"id"`

	APIKey string `json:"apiKey"`

	LastConnected time.Time    `json:"lastConnected"`
	User          *itchio.User `json:"user"`
	UserID        int64        `json:"userId"`

	Developer bool `json:"developer"`
	PressUser bool `json:"pressUser"`

	Collections  []*itchio.Collection  `json:"collections,omitempty" gorm:"many2many:profile_collections"`
	ProfileGames []*ProfileGame        `json:"profileGames,omitempty"`
	OwnedKeys    []*itchio.DownloadKey `json:"ownedKeys,omitempty" gorm:"foreignKey:owner_id"`
}

func (p *Profile) UpdateFromUser(user *itchio.User) {
	p.User = user
	p.Developer = user.Developer
	p.PressUser = user.PressUser
	p.LastConnected = time.Now().UTC()
}
