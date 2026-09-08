// Package steam holds the Steam side of butler's sync feature: stored
// credentials, login, and the publisher key check. It is shared by the
// CLI commands and the butlerd endpoints, so it reports through return
// values and callbacks rather than printing.
//
// Two Steam credentials are involved. A refresh token from a regular
// account login is what the depot protocol needs to download files, but
// it says nothing about who made the game. A publisher Web API key from
// the partner site proves which apps the developer controls, so every
// download is gated on it.
package steam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/itchio/fresh-steamer/partner"
	"github.com/itchio/fresh-steamer/session"
)

var (
	ErrNotLoggedIn         = errors.New("not logged in to Steam")
	ErrNoPublisherKey      = errors.New("no Steam publisher key stored")
	ErrPublisherKeyInvalid = errors.New("Steam rejected the publisher key")
)

// ErrAppNotControlled means the publisher key does not list the app.
type ErrAppNotControlled struct {
	AppID uint32
}

func (e *ErrAppNotControlled) Error() string {
	return fmt.Sprintf("app %d is not in the list of apps your Steam publisher key controls", e.AppID)
}

type Creds struct {
	AccountName  string `json:"account_name,omitempty"`
	SteamID      uint64 `json:"steam_id,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	PublisherKey string `json:"publisher_key,omitempty"`
}

func (c *Creds) LoggedIn() bool        { return c.RefreshToken != "" }
func (c *Creds) HasPublisherKey() bool { return c.PublisherKey != "" }

// overlay replaces the login as a unit when o carries a token, since a
// token only works with the account name it was issued to, and the key
// on its own.
func (c *Creds) overlay(o *Creds) {
	if o.RefreshToken != "" {
		c.RefreshToken = o.RefreshToken
		c.AccountName = o.AccountName
		c.SteamID = o.SteamID
	}
	if o.PublisherKey != "" {
		c.PublisherKey = o.PublisherKey
	}
}

// merge fills in each field of c from o independently.
func (c *Creds) merge(o *Creds) {
	if o == nil {
		return
	}
	if o.RefreshToken != "" {
		c.RefreshToken = o.RefreshToken
	}
	if o.AccountName != "" {
		c.AccountName = o.AccountName
	}
	if o.SteamID != 0 {
		c.SteamID = o.SteamID
	}
	if o.PublisherKey != "" {
		c.PublisherKey = o.PublisherKey
	}
}

// Environment variables that supply credentials without a file, for CI
// and for callers that hold a login in memory and spawn butler.
const (
	EnvRefreshToken = "BUTLER_STEAM_REFRESH_TOKEN"
	EnvAccountName  = "BUTLER_STEAM_ACCOUNT_NAME"
	EnvPublisherKey = "BUTLER_STEAM_PUBLISHER_KEY"
)

func fromEnv() *Creds {
	return &Creds{
		AccountName:  os.Getenv(EnvAccountName),
		RefreshToken: os.Getenv(EnvRefreshToken),
		PublisherKey: os.Getenv(EnvPublisherKey),
	}
}

// Store is where Steam state lives on disk: next to butler_creds, so
// `-i` moves it along with the itch.io identity and the CLI and the app
// see the same login.
//
// Reads see the file underneath whatever the caller gives through the
// environment and Override. Those two combine field by field, so a token
// from one and an account name from the other form one login, which then
// replaces the file's login as a unit. Writes only ever touch the file,
// so a login given on the command line never lands on disk through an
// unrelated save such as storing a publisher key.
type Store struct {
	Dir string
	// Override is per-call, for credentials given as flags.
	Override *Creds
}

func StoreFor(identityPath string) Store {
	return Store{Dir: filepath.Dir(identityPath)}
}

func (s Store) CredsPath() string { return filepath.Join(s.Dir, "steam_creds.json") }
func (s Store) KeysPath() string  { return filepath.Join(s.Dir, "steam_depot_keys.json") }

// Load returns the effective credentials, empty when there are none.
func (s Store) Load() (*Creds, error) {
	c, err := s.Persisted()
	if err != nil {
		return nil, err
	}
	given := fromEnv()
	given.merge(s.Override)
	if given.RefreshToken != "" && given.AccountName == "" {
		return nil, fmt.Errorf("a Steam refresh token was given without an account name (%s or --steam-account-name)", EnvAccountName)
	}
	c.overlay(given)
	return c, nil
}

// Persisted returns only what is in the file.
func (s Store) Persisted() (*Creds, error) {
	data, err := os.ReadFile(s.CredsPath())
	if os.IsNotExist(err) {
		return &Creds{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading steam credentials: %w", err)
	}
	var c Creds
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing steam credentials: %w", err)
	}
	return &c, nil
}

func (s Store) Save(c *Creds) error {
	p := s.CredsPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return fmt.Errorf("writing steam credentials: %w", err)
	}
	return nil
}

// Update applies f to the persisted credentials and saves the result.
func (s Store) Update(f func(c *Creds)) (*Creds, error) {
	c, err := s.Persisted()
	if err != nil {
		return nil, err
	}
	f(c)
	if err := s.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

// Logout removes the login, the publisher key and the cached depot keys.
func (s Store) Logout() error {
	for _, p := range []string{s.CredsPath(), s.KeysPath()} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (s Store) RequireLogin() (*Creds, error) {
	c, err := s.Load()
	if err != nil {
		return nil, err
	}
	if !c.LoggedIn() {
		return nil, ErrNotLoggedIn
	}
	return c, nil
}

func (s Store) PartnerClient() (*partner.Client, error) {
	c, err := s.Load()
	if err != nil {
		return nil, err
	}
	if !c.HasPublisherKey() {
		return nil, ErrNoPublisherKey
	}
	return newPartnerClient(c.PublisherKey), nil
}

// BUTLER_STEAM_PARTNER_URL points the partner API at a stand-in server,
// which is how the integration tests exercise the key flow.
func newPartnerClient(key string) *partner.Client {
	pc := partner.NewClient(key)
	if u := os.Getenv("BUTLER_STEAM_PARTNER_URL"); u != "" {
		pc.BaseURL = u
	}
	return pc
}

// OpenSession connects to Steam with the stored login. logf may be nil.
func (s Store) OpenSession(ctx context.Context, logf func(string, ...interface{})) (*session.Session, error) {
	c, err := s.RequireLogin()
	if err != nil {
		return nil, err
	}
	if logf == nil {
		logf = func(string, ...interface{}) {}
	}
	sess, err := session.Open(ctx, session.Options{
		AccountName:  c.AccountName,
		RefreshToken: c.RefreshToken,
		KeyFile:      s.KeysPath(),
		Logf:         logf,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to Steam: %w", err)
	}
	return sess, nil
}

// CheckAppAccess refuses apps the publisher key does not control. The
// depot protocol would happily serve any owned game, and this tool is for
// developers moving their own builds, not for copying a library.
func (s Store) CheckAppAccess(ctx context.Context, appID uint32) error {
	if Ungated() {
		return nil
	}
	pc, err := s.PartnerClient()
	if err != nil {
		return err
	}
	ok, err := pc.HasApp(ctx, appID)
	if err != nil {
		return wrapPartnerErr(err, "checking app access with publisher key")
	}
	if !ok {
		return &ErrAppNotControlled{AppID: appID}
	}
	return nil
}

func wrapPartnerErr(err error, what string) error {
	var se *partner.StatusError
	if errors.As(err, &se) && se.Unauthorized() {
		return fmt.Errorf("%s: %w", what, ErrPublisherKeyInvalid)
	}
	return fmt.Errorf("%s: %w", what, err)
}
