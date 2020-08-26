package profile

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/itchio/httpkit/neterr"
	"github.com/pkg/errors"
	"xorm.io/builder"
)

func Register(router *butlerd.Router) {
	messages.ProfileList.Register(router, List)
	messages.ProfileLoginWithPassword.Register(router, LoginWithPassword)
	messages.ProfileLoginWithAPIKey.Register(router, LoginWithAPIKey)
	messages.ProfileUseSavedLogin.Register(router, UseSavedLogin)
	messages.ProfileForget.Register(router, Forget)
	messages.ProfileDataPut.Register(router, DataPut)
	messages.ProfileDataGet.Register(router, DataGet)
}

func List(rc *butlerd.RequestContext, params butlerd.ProfileListParams) (*butlerd.ProfileListResult, error) {
	var profiles []*models.Profile
	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustSelect(conn, &profiles, builder.NewCond(), hades.Search{})
		models.MustPreload(conn, profiles, hades.Assoc("User"))
	})

	var formattedProfiles []*butlerd.Profile
	for _, profile := range profiles {
		formattedProfiles = append(formattedProfiles, formatProfile(profile))
	}

	return &butlerd.ProfileListResult{
		Profiles: formattedProfiles,
	}, nil
}

func formatProfile(p *models.Profile) *butlerd.Profile {
	return &butlerd.Profile{
		ID:            p.ID,
		LastConnected: p.LastConnected,
		User:          p.User,
	}
}

func LoginWithPassword(rc *butlerd.RequestContext, params butlerd.ProfileLoginWithPasswordParams) (*butlerd.ProfileLoginWithPasswordResult, error) {
	if params.Username == "" {
		return nil, errors.New("username cannot be empty")
	}
	if params.Password == "" {
		return nil, errors.New("password cannot be empty")
	}

	rootClient := rc.RootClient()

	var key *itchio.APIKey
	var cookie itchio.Cookie

	{
		loginRes, err := rootClient.LoginWithPassword(rc.Ctx, itchio.LoginWithPasswordParams{
			Username:       params.Username,
			Password:       params.Password,
			ForceRecaptcha: params.ForceRecaptcha,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if loginRes.RecaptchaNeeded {
			// Captcha flow
			recaptchaRes, err := messages.ProfileRequestCaptcha.Call(rc, butlerd.ProfileRequestCaptchaParams{
				RecaptchaURL: loginRes.RecaptchaURL,
			})
			if err != nil {
				return nil, errors.WithStack(err)
			}

			if recaptchaRes.RecaptchaResponse == "" {
				return nil, errors.WithStack(butlerd.CodeOperationAborted)
			}

			loginRes, err = rootClient.LoginWithPassword(rc.Ctx, itchio.LoginWithPasswordParams{
				Username:          params.Username,
				Password:          params.Password,
				RecaptchaResponse: recaptchaRes.RecaptchaResponse,
			})
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}

		if loginRes.Token != "" {
			// TOTP flow
			totpRes, err := messages.ProfileRequestTOTP.Call(rc, butlerd.ProfileRequestTOTPParams{})
			if err != nil {
				return nil, errors.WithStack(err)
			}

			if totpRes.Code == "" {
				return nil, errors.WithStack(butlerd.CodeOperationAborted)
			}

			verifyRes, err := rootClient.TOTPVerify(rc.Ctx, itchio.TOTPVerifyParams{
				Token: loginRes.Token,
				Code:  totpRes.Code,
			})
			if err != nil {
				return nil, errors.WithStack(err)
			}

			key = verifyRes.Key
			cookie = verifyRes.Cookie
		} else {
			// One-factor flow
			key = loginRes.Key
			cookie = loginRes.Cookie
		}
	}

	client := rc.Client(key.Key)

	profileRes, err := client.GetProfile(rc.Ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	profile := &models.Profile{
		ID:     profileRes.User.ID,
		APIKey: key.Key,
	}
	profile.UpdateFromUser(profileRes.User)
	rc.WithConn(profile.Save)

	res := &butlerd.ProfileLoginWithPasswordResult{
		Cookie:  cookie,
		Profile: formatProfile(profile),
	}
	return res, nil
}

func LoginWithAPIKey(rc *butlerd.RequestContext, params butlerd.ProfileLoginWithAPIKeyParams) (*butlerd.ProfileLoginWithAPIKeyResult, error) {
	if params.APIKey == "" {
		return nil, errors.New("apiKey cannot be empty")
	}

	client := rc.Client(params.APIKey)

	profileRes, err := client.GetProfile(rc.Ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	profile := &models.Profile{
		ID:     profileRes.User.ID,
		APIKey: params.APIKey,
	}
	profile.UpdateFromUser(profileRes.User)
	rc.WithConn(profile.Save)

	res := &butlerd.ProfileLoginWithAPIKeyResult{
		Profile: formatProfile(profile),
	}
	return res, nil
}

func UseSavedLogin(rc *butlerd.RequestContext, params butlerd.ProfileUseSavedLoginParams) (*butlerd.ProfileUseSavedLoginResult, error) {
	consumer := rc.Consumer

	profile, client := rc.ProfileClient(params.ProfileID)

	consumer.Opf("Validating credentials...")

	err := func() error {
		profileRes, err := client.GetProfile(rc.Ctx)
		if err != nil {
			return errors.WithStack(err)
		}

		profile.UpdateFromUser(profileRes.User)
		rc.WithConn(profile.Save)

		return nil
	}()
	if err != nil {
		if neterr.IsNetworkError(err) {
			rc.WithConn(func(conn *sqlite.Conn) {
				err := models.Preload(conn, profile, hades.Assoc("User"))
				if err != nil {
					consumer.Warnf("Could not preload user on profile: %+v", err)
				}
			})
			if profile.User == nil {
				consumer.Warnf("Could not perform offline login...")
				return nil, err
			}
			consumer.Opf("Logged in! (offline)")
		} else {
			return nil, err
		}
	} else {
		consumer.Opf("Logged in! (online)")
	}

	res := &butlerd.ProfileUseSavedLoginResult{
		Profile: formatProfile(profile),
	}
	return res, nil
}

func Forget(rc *butlerd.RequestContext, params butlerd.ProfileForgetParams) (*butlerd.ProfileForgetResult, error) {
	if params.ProfileID == 0 {
		return nil, errors.New("profileId must be set")
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustDelete(conn, &models.Profile{}, builder.Eq{"id": params.ProfileID})
	})

	res := &butlerd.ProfileForgetResult{
		Success: true,
	}
	return res, nil
}
