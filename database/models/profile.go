package models

import (
	"time"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
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
	OwnedKeys          []*itchio.DownloadKey `json:"ownedKeys,omitempty" hades:"foreign_key:owner_id"`
}

func (p *Profile) UpdateFromUser(user *itchio.User) {
	p.User = user
	p.Developer = user.Developer
	p.PressUser = user.PressUser
	p.LastConnected = time.Now().UTC()
}

func (p *Profile) Save(conn *sqlite.Conn) {
	MustSave(conn, p,
		hades.Assoc("User"),
	)
}

func ProfileByID(conn *sqlite.Conn, id int64) *Profile {
	var p Profile
	if MustSelectOne(conn, &p, builder.Eq{"id": id}) {
		return &p
	}
	return nil
}

func AllProfiles(conn *sqlite.Conn) []*Profile {
	var profiles []*Profile
	MustSelect(conn, &profiles, builder.NewCond(), nil)
	return profiles
}
