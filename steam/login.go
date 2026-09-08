package steam

import (
	"context"
	"fmt"

	"github.com/itchio/fresh-steamer/auth"
)

type LoginOptions struct {
	// Persist writes the login to the file. It is returned either way.
	Persist bool
}

// LoginQR runs the QR login. onChallenge receives the URL to show as a
// QR code; Steam rotates it, so it may be called more than once. A
// publisher key already on file is kept.
func LoginQR(ctx context.Context, s Store, onChallenge func(url string), opts LoginOptions) (*Creds, error) {
	c, err := auth.LoginQR(ctx, auth.QROptions{OnChallenge: onChallenge})
	if err != nil {
		return nil, fmt.Errorf("logging in to Steam: %w", err)
	}
	return remember(s, c, opts)
}

// LoginPassword logs in with account name and password. guard is asked
// for Steam Guard codes; for mobile confirmation it is called with
// auth.GuardDeviceConfirmation and should return "" after telling the
// user to approve on their phone.
func LoginPassword(ctx context.Context, s Store, accountName, password string, guard auth.Guard, opts LoginOptions) (*Creds, error) {
	c, err := auth.Login(ctx, auth.Options{
		AccountName: accountName,
		Password:    password,
		Guard:       guard,
	})
	if err != nil {
		return nil, fmt.Errorf("logging in to Steam: %w", err)
	}
	return remember(s, c, opts)
}

func remember(s Store, c *auth.Credentials, opts LoginOptions) (*Creds, error) {
	login := &Creds{
		AccountName:  c.AccountName,
		SteamID:      c.SteamID,
		RefreshToken: c.RefreshToken,
	}
	if opts.Persist {
		_, err := s.Update(func(cr *Creds) {
			cr.AccountName = login.AccountName
			cr.SteamID = login.SteamID
			cr.RefreshToken = login.RefreshToken
		})
		if err != nil {
			return nil, err
		}
	}
	return login, nil
}
