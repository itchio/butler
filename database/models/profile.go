package models

import (
	"time"

	itchio "github.com/itchio/go-itchio"
)

type Profile struct {
	ID int64 `json:"id"`

	APIKey string `json:"apiKey"`

	LastConnected time.Time `json:"lastConnected"`
	User          JSON      `json:"user"`
}

func (p *Profile) SetUser(user *itchio.User) error { return MarshalUser(user, &p.User) }
func (p *Profile) GetUser() (*itchio.User, error)  { return UnmarshalUser(p.User) }
