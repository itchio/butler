package steam

import (
	"context"
	"fmt"

	"github.com/itchio/fresh-steamer/auth"
)

// Account is what a login yields, minus the token.
type Account struct {
	AccountName string
	SteamID     uint64
}

// LoginQR runs the QR login and saves the result. onChallenge receives
// the URL to show as a QR code; Steam rotates it, so it may be called
// more than once. The publisher key, if any, is kept.
func LoginQR(ctx context.Context, s Store, onChallenge func(url string)) (*Account, error) {
	c, err := auth.LoginQR(ctx, auth.QROptions{OnChallenge: onChallenge})
	if err != nil {
		return nil, fmt.Errorf("logging in to Steam: %w", err)
	}
	return save(s, c)
}

// LoginPassword logs in with account name and password. guard is asked
// for Steam Guard codes; for mobile confirmation it is called with
// auth.GuardDeviceConfirmation and should return "" after telling the
// user to approve on their phone.
func LoginPassword(ctx context.Context, s Store, accountName, password string, guard auth.Guard) (*Account, error) {
	c, err := auth.Login(ctx, auth.Options{
		AccountName: accountName,
		Password:    password,
		Guard:       guard,
	})
	if err != nil {
		return nil, fmt.Errorf("logging in to Steam: %w", err)
	}
	return save(s, c)
}

func save(s Store, c *auth.Credentials) (*Account, error) {
	_, err := s.Update(func(cr *Creds) {
		cr.AccountName = c.AccountName
		cr.SteamID = c.SteamID
		cr.RefreshToken = c.RefreshToken
	})
	if err != nil {
		return nil, err
	}
	return &Account{AccountName: c.AccountName, SteamID: c.SteamID}, nil
}
