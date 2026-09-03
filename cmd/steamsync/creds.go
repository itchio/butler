// Package steamsync holds the butler commands that move a developer's
// Steam builds and store listing over to itch.io, backed by
// github.com/itchio/fresh-steamer.
//
// Two Steam credentials are involved. A refresh token from a regular
// account login is what the depot protocol needs to download files, but
// it says nothing about who made the game. A publisher Web API key from
// the partner site proves which apps the developer controls, so every
// download is gated on it.
package steamsync

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/fresh-steamer/partner"
	"github.com/itchio/fresh-steamer/session"
	"github.com/pkg/errors"
	"golang.org/x/term"
)

type creds struct {
	AccountName  string `json:"account_name,omitempty"`
	SteamID      uint64 `json:"steam_id,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	PublisherKey string `json:"publisher_key,omitempty"`
}

// Steam state lives next to butler_creds so `-i` moves it along with the
// itch.io identity.
func credsPath(ctx *mansion.Context) string {
	return filepath.Join(filepath.Dir(ctx.Identity), "steam_creds.json")
}

func keysPath(ctx *mansion.Context) string {
	return filepath.Join(filepath.Dir(ctx.Identity), "steam_depot_keys.json")
}

func loadCreds(ctx *mansion.Context) (*creds, error) {
	data, err := os.ReadFile(credsPath(ctx))
	if os.IsNotExist(err) {
		return &creds{}, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "reading steam credentials")
	}
	var c creds
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, errors.Wrap(err, "parsing steam credentials")
	}
	return &c, nil
}

func saveCreds(ctx *mansion.Context, c *creds) error {
	p := credsPath(ctx)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return errors.Wrap(err, "creating config directory")
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return errors.Wrap(err, "writing steam credentials")
	}
	return nil
}

func requireLogin(ctx *mansion.Context) (*creds, error) {
	c, err := loadCreds(ctx)
	if err != nil {
		return nil, err
	}
	if c.RefreshToken == "" {
		return nil, errors.New("not logged in to Steam, run `butler steam-login` first")
	}
	return c, nil
}

func requireKey(ctx *mansion.Context) (*creds, error) {
	c, err := loadCreds(ctx)
	if err != nil {
		return nil, err
	}
	if c.PublisherKey == "" {
		return nil, errors.New("no Steam publisher key stored, run `butler steam-key` first")
	}
	return c, nil
}

func partnerClient(ctx *mansion.Context) (*partner.Client, error) {
	c, err := requireKey(ctx)
	if err != nil {
		return nil, err
	}
	return partner.NewClient(c.PublisherKey), nil
}

func openSession(ctx *mansion.Context, goCtx context.Context) (*session.Session, error) {
	c, err := requireLogin(ctx)
	if err != nil {
		return nil, err
	}
	s, err := session.Open(goCtx, session.Options{
		AccountName:  c.AccountName,
		RefreshToken: c.RefreshToken,
		KeyFile:      keysPath(ctx),
		Logf:         comm.Debugf,
	})
	if err != nil {
		return nil, errors.Wrap(err, "connecting to Steam")
	}
	return s, nil
}

// checkAppAccess refuses apps the publisher key does not control. The
// depot protocol would happily serve any owned game, and this tool is for
// developers moving their own builds, not for copying a library.
func checkAppAccess(ctx *mansion.Context, goCtx context.Context, appID uint32) error {
	pc, err := partnerClient(ctx)
	if err != nil {
		return err
	}
	ok, err := pc.HasApp(goCtx, appID)
	if err != nil {
		return errors.Wrap(err, "checking app access with publisher key")
	}
	if !ok {
		return errors.Errorf("app %d is not in the list of apps your Steam publisher key controls", appID)
	}
	return nil
}

func prompt(label string, secret bool) (string, error) {
	fmt.Fprint(os.Stderr, label)
	if secret && term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		return string(b), err
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
