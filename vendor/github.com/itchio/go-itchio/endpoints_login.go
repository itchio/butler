package itchio

//-------------------------------------------------------

// LoginWithPasswordParams : params for LoginWithPassword
type LoginWithPasswordParams struct {
	Username          string
	Password          string
	RecaptchaResponse string
}

// Cookie represents, well, multiple key=value pairs that
// should be set to obtain a logged-in browser session for
// the user who just logged in.
type Cookie map[string]string

// LoginWithPasswordResponse : response for LoginWithPassword
type LoginWithPasswordResponse struct {
	RecaptchaNeeded bool   `json:"recaptchaNeeded"`
	RecaptchaURL    string `json:"recaptchaUrl"`
	TOTPNeeded      bool   `json:"totpNeeded"`
	Token           string `json:"token"`

	Key    *APIKey `json:"key"`
	Cookie Cookie  `json:"cookie"`
}

// LoginWithPassword attempts to log a user into itch.io with
// their username (or e-mail) and password.
// The response may indicate that a TOTP code is needed (for two-factor auth),
// or a recaptcha challenge is needed (an unfortunate remedy for an unfortunate ailment).
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

// TOTPVerifyParams : params for TOTPVerify
type TOTPVerifyParams struct {
	Token string
	Code  string
}

// TOTPVerifyResponse : response for TOTPVerify
type TOTPVerifyResponse struct {
	Key    *APIKey `json:"key"`
	Cookie Cookie  `json:"cookie"`
}

// TOTPVerify sends a user-entered TOTP token to the server for
// verification (and to complete login).
func (c *Client) TOTPVerify(params TOTPVerifyParams) (*TOTPVerifyResponse, error) {
	q := NewQuery(c, "/totp/verify")
	q.AddString("token", params.Token)
	q.AddString("code", params.Code)

	r := &TOTPVerifyResponse{}
	return r, q.Post(r)
}

//-------------------------------------------------------

// SubkeyParams : params for Subkey
type SubkeyParams struct {
	GameID int64
	Scope  string
}

// SubkeyResponse : params for Subkey
type SubkeyResponse struct {
	Key       string `json:"key"`
	ExpiresAt string `json:"expiresAt"`
}

// Subkey creates a scoped-down, temporary offspring of the main
// API key this client was created with. It is useful to automatically grant
// some access to games being launched.
func (c *Client) Subkey(params SubkeyParams) (*SubkeyResponse, error) {
	q := NewQuery(c, "/credentials/subkey")
	q.AddInt64("game_id", params.GameID)
	q.AddString("scope", params.Scope)

	r := &SubkeyResponse{}
	return r, q.Post(r)
}
