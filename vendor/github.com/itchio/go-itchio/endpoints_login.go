package itchio

import (
	"fmt"
	"net/url"

	"github.com/pkg/errors"
)

//-------------------------------------------------------

type LoginWithPasswordParams struct {
	Username          string
	Password          string
	RecaptchaResponse string
}

type Cookie map[string]string

type LoginWithPasswordResponse struct {
	RecaptchaNeeded bool   `json:"recaptchaNeeded"`
	RecaptchaURL    string `json:"recaptchaUrl"`
	TOTPNeeded      bool   `json:"totpNeeded"`
	Token           string `json:"token"`

	Key    *APIKey `json:"key"`
	Cookie Cookie  `json:"cookie"`
}

func (c *Client) LoginWithPassword(params *LoginWithPasswordParams) (*LoginWithPasswordResponse, error) {
	r := &LoginWithPasswordResponse{}
	path := c.MakePath("/login")

	form := url.Values{}
	form.Add("source", "desktop")
	form.Add("username", params.Username)
	form.Add("password", params.Password)
	if params.RecaptchaResponse != "" {
		form.Add("recaptcha_response", params.RecaptchaResponse)
	}

	err := c.PostFormResponse(path, form, r)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return r, nil
}

//-------------------------------------------------------

type TOTPVerifyParams struct {
	Token string
	Code  string
}

type TOTPVerifyResponse struct {
	Key    *APIKey `json:"key"`
	Cookie Cookie  `json:"cookie"`
}

func (c *Client) TOTPVerify(params *TOTPVerifyParams) (*TOTPVerifyResponse, error) {
	r := &TOTPVerifyResponse{}
	path := c.MakePath("/totp/verify")

	form := url.Values{}
	form.Add("token", params.Token)
	form.Add("code", params.Code)

	err := c.PostFormResponse(path, form, r)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return r, nil
}

//-------------------------------------------------------

type SubkeyParams struct {
	GameID int64
	Scope  string
}

type SubkeyResponse struct {
	Key       string `json:"key"`
	ExpiresAt string `json:"expiresAt"`
}

func (c *Client) Subkey(params *SubkeyParams) (*SubkeyResponse, error) {
	r := &SubkeyResponse{}
	path := c.MakePath("/credentials/subkey")

	form := url.Values{}
	form.Add("game_id", fmt.Sprintf("%d", params.GameID))
	form.Add("scope", params.Scope)

	err := c.PostFormResponse(path, form, r)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return r, nil
}
