package session

import (
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/go-itchio"
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
	rootClient, err := rc.RootClient()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var key *itchio.APIKey
	var cookie itchio.Cookie

	{
		loginRes, err := rootClient.LoginWithPassword(&itchio.LoginWithPasswordParams{
			Username: params.Username,
			Password: params.Password,
		})
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		if loginRes.RecaptchaNeeded {
			// Captcha flow
			recaptchaRes, err := messages.SessionRequestCaptcha.Call(rc, &buse.SessionRequestCaptchaParams{
				RecaptchaURL: loginRes.RecaptchaURL,
			})
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}

			if recaptchaRes.RecaptchaResponse == "" {
				return nil, &buse.ErrAborted{}
			}

			loginRes, err = rootClient.LoginWithPassword(&itchio.LoginWithPasswordParams{
				Username:          params.Username,
				Password:          params.Password,
				RecaptchaResponse: recaptchaRes.RecaptchaResponse,
			})
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}
		}

		if loginRes.Token != "" {
			// TOTP flow
			totpRes, err := messages.SessionRequestTOTP.Call(rc, &buse.SessionRequestTOTPParams{})
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}

			if totpRes.Code == "" {
				return nil, &buse.ErrAborted{}
			}

			verifyRes, err := rootClient.TOTPVerify(&itchio.TOTPVerifyParams{
				Token: loginRes.Token,
				Code:  totpRes.Code,
			})
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}

			key = verifyRes.Key
			cookie = verifyRes.Cookie
		} else {
			// One-factor flow
			key = loginRes.Key
			cookie = loginRes.Cookie
		}
	}

	client, err := rc.KeyClient(key.Key)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	meRes, err := client.GetMe()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	profile := &models.Profile{
		ID:            meRes.User.ID,
		APIKey:        key.Key,
		LastConnected: time.Now().UTC(),
	}
	err = profile.SetUser(meRes.User)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = db.Save(profile).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	session, err := profileToSession(profile)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.SessionLoginWithPasswordResult{
		Cookie:  cookie,
		Session: session,
	}
	return res, nil
}

func UseSavedLogin(rc *buse.RequestContext, params *buse.SessionUseSavedLoginParams) (*buse.SessionUseSavedLoginResult, error) {
	if params.SessionID == 0 {
		return nil, errors.New("sessionID must be non-zero")
	}

	consumer := rc.Consumer
	consumer.Opf("Fetching saved info...")

	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	profile := &models.Profile{}
	err = db.Where("id = ?", params.SessionID).First(profile).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Opf("Validating credentials...")

	client, err := rc.MansionContext.NewClient(profile.APIKey)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	meRes, err := client.GetMe()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	profile.LastConnected = time.Now().UTC()
	err = profile.SetUser(meRes.User)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = db.Save(profile).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	session, err := profileToSession(profile)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Opf("Logged in!")

	res := &buse.SessionUseSavedLoginResult{
		Session: session,
	}
	return res, nil
}

func Forget(rc *buse.RequestContext, params *buse.SessionForgetParams) (*buse.SessionForgetResult, error) {
	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = db.Where("id = ?", params.SessionID).Delete(&models.Profile{}).Error
	success := db.RowsAffected > 1
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.SessionForgetResult{
		Success: success,
	}
	return res, nil
}
