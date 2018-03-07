package models

import (
	"time"

	"github.com/go-errors/errors"
	itchio "github.com/itchio/go-itchio"
	"github.com/jinzhu/gorm"
)

type Profile struct {
	ID int64 `json:"id"`

	APIKey string `json:"apiKey"`

	LastConnected time.Time    `json:"lastConnected"`
	User          *itchio.User `json:"user"`
	UserID        int64        `json:"userId"`

	Developer bool `json:"developer"`
	PressUser bool `json:"pressUser"`

	ProfileCollections []*ProfileCollection  `json:"profileCollections,omitempty"`
	ProfileGames       []*ProfileGame        `json:"profileGames,omitempty"`
	OwnedKeys          []*itchio.DownloadKey `json:"ownedKeys,omitempty" gorm:"foreignKey:owner_id"`
}

func (p *Profile) UpdateFromUser(user *itchio.User) {
	p.User = user
	p.Developer = user.Developer
	p.PressUser = user.PressUser
	p.LastConnected = time.Now().UTC()
}

func ProfileByID(db *gorm.DB, id int64) (*Profile, error) {
	p := &Profile{}
	req := db.Where("id = ?", id).First(p)
	if req.Error != nil {
		if req.RecordNotFound() {
			return nil, nil
		}
		return nil, errors.Wrap(req.Error, 0)
	}
	return p, nil
}

func AllProfiles(db *gorm.DB) ([]*Profile, error) {
	var profiles []*Profile
	err := db.Find(&profiles).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return profiles, nil
}
