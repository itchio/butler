package session

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/database/models"
)

func Register(router *buse.Router) {
	messages.SessionList.Register(router, List)
	messages.SessionLoginWithPassword.Register(router, LoginWithPassword)
	messages.SessionUseSavedLogin.Register(router, UseSavedLogin)
	messages.SessionForget.Register(router, Forget)
}

func List(rc *buse.RequestContext, params *buse.SessionListParams) (*buse.SessionListResult, error) {
	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var profiles []*models.Profile
	err = db.Order("last_connected desc").Find(&profiles).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var sessions []*buse.Session
	for _, profile := range profiles {
		p, err := profileToSession(profile)
		if err != nil {
			rc.Consumer.Warnf(err.Error())
			rc.Consumer.Warnf("...skipping one profile")
			return nil, errors.Wrap(err, 0)
		}
		sessions = append(sessions, p)
	}

	return &buse.SessionListResult{
		Sessions: sessions,
	}, nil
}

func profileToSession(p *models.Profile) (*buse.Session, error) {
	user, err := p.GetUser()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	s := &buse.Session{
		ID:            p.ID,
		LastConnected: p.LastConnected,
		User:          user,
	}
	return s, nil
}

func LoginWithPassword(rc *buse.RequestContext, params *buse.SessionLoginWithPasswordParams) (*buse.SessionLoginWithPasswordResult, error) {
	return nil, errors.New("stub!")
}

func UseSavedLogin(rc *buse.RequestContext, params *buse.SessionUseSavedLoginParams) (*buse.SessionUseSavedLoginResult, error) {
	return nil, errors.New("stub!")
}

func Forget(rc *buse.RequestContext, params *buse.SessionForgetParams) (*buse.SessionForgetResult, error) {
	return nil, errors.New("stub!")
}
