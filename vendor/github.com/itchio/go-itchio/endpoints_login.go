package itchio

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

func (c *Client) LoginWithPassword(params LoginWithPasswordParams) (*LoginWithPasswordResponse, error) {
	q := NewQuery(c, "/login")
	q.AddString("source", "desktop")
	q.AddString("username", params.Username)
	q.AddString("password", params.Password)
	q.AddStringIfNonEmpty("recaptcha_response", params.RecaptchaResponse)

	r := &LoginWithPasswordResponse{}
	return r, q.Post(r)
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

func (c *Client) TOTPVerify(params TOTPVerifyParams) (*TOTPVerifyResponse, error) {
	q := NewQuery(c, "/totp/verify")
	q.AddString("token", params.Token)
	q.AddString("code", params.Code)

	r := &TOTPVerifyResponse{}
	return r, q.Post(r)
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

func (c *Client) Subkey(params SubkeyParams) (*SubkeyResponse, error) {
	q := NewQuery(c, "/credentials/subkey")
	q.AddInt64("game_id", params.GameID)
	q.AddString("scope", params.Scope)

	r := &SubkeyResponse{}
	return r, q.Post(r)
}
