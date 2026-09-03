package steamsync

import (
	"context"
	"fmt"
	"os"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/fresh-steamer/auth"
	"github.com/itchio/fresh-steamer/partner"
	"github.com/mdp/qrterminal/v3"
	"github.com/pkg/errors"
)

var loginArgs = struct {
	password bool
	user     string
}{}

var keyArgs = struct {
	key string
}{}

func RegisterLogin(ctx *mansion.Context) {
	cmd := ctx.App.Command("steam-login", "Log in to a Steam account so butler can download your builds.").Hidden()
	cmd.Flag("password", "Log in with account name and password instead of scanning a QR code").BoolVar(&loginArgs.password)
	cmd.Flag("user", "Steam account name (password login only)").StringVar(&loginArgs.user)
	ctx.Register(cmd, doLogin)

	logout := ctx.App.Command("steam-logout", "Remove saved Steam credentials and publisher key.").Hidden()
	ctx.Register(logout, doLogout)

	key := ctx.App.Command("steam-key", "Store a Steam publisher Web API key. It proves which apps you control.").Hidden()
	key.Arg("key", "The key. Prompted for when omitted.").StringVar(&keyArgs.key)
	ctx.Register(key, doKey)
}

func doLogin(ctx *mansion.Context) {
	ctx.Must(Login(ctx, loginArgs.password, loginArgs.user))
}

func Login(ctx *mansion.Context, usePassword bool, user string) error {
	goCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	existing, err := loadCreds(ctx)
	if err != nil {
		return err
	}
	if existing.RefreshToken != "" {
		comm.Logf("Already logged in to Steam as %s. Run `butler steam-logout` to switch accounts.", existing.AccountName)
		comm.Result(map[string]string{"status": "success", "account_name": existing.AccountName})
		return nil
	}

	var c *auth.Credentials
	if usePassword {
		c, err = loginPassword(goCtx, user)
	} else {
		c, err = loginQR(goCtx)
	}
	if err != nil {
		return errors.Wrap(err, "logging in to Steam")
	}

	existing.AccountName = c.AccountName
	existing.SteamID = c.SteamID
	existing.RefreshToken = c.RefreshToken
	if err := saveCreds(ctx, existing); err != nil {
		return err
	}
	comm.Logf("Logged in to Steam as %s, credentials saved to %s", c.AccountName, credsPath(ctx))
	if existing.PublisherKey == "" {
		comm.Logf("Next, run `butler steam-key` to store the publisher key that proves which apps you control.")
	}
	comm.Result(map[string]string{"status": "success", "account_name": c.AccountName})
	return nil
}

func loginQR(goCtx context.Context) (*auth.Credentials, error) {
	return auth.LoginQR(goCtx, auth.QROptions{
		OnChallenge: func(url string) {
			comm.Logf("")
			comm.Logf("Scan with the Steam mobile app, then approve the login there.")
			comm.Logf("Or open this link on your phone: %s", url)
			comm.Logf("")
			qrterminal.GenerateWithConfig(url, qrterminal.Config{
				Level:          qrterminal.L,
				Writer:         os.Stderr,
				HalfBlocks:     true,
				BlackChar:      qrterminal.BLACK_BLACK,
				WhiteChar:      qrterminal.WHITE_WHITE,
				BlackWhiteChar: qrterminal.BLACK_WHITE,
				WhiteBlackChar: qrterminal.WHITE_BLACK,
				QuietZone:      2,
			})
			comm.Logf("")
			comm.Logf("Waiting for approval... (ctrl-c to cancel, or use `butler steam-login --password`)")
		},
	})
}

func loginPassword(goCtx context.Context, name string) (*auth.Credentials, error) {
	var err error
	if name == "" {
		if name, err = prompt("Steam account name: ", false); err != nil {
			return nil, err
		}
	}
	pass, err := prompt("Steam password: ", true)
	if err != nil {
		return nil, err
	}
	return auth.Login(goCtx, auth.Options{
		AccountName: name,
		Password:    pass,
		Guard: auth.GuardFunc(func(ctx context.Context, kind auth.GuardType, msg string) (string, error) {
			if kind == auth.GuardDeviceConfirmation {
				comm.Logf("Approve the login in the Steam mobile app...")
				return "", nil
			}
			label := fmt.Sprintf("Steam Guard %s", kind)
			if msg != "" {
				label += " (" + msg + ")"
			}
			return prompt(label+": ", false)
		}),
	})
}

func doLogout(ctx *mansion.Context) {
	ctx.Must(Logout(ctx))
}

func Logout(ctx *mansion.Context) error {
	p := credsPath(ctx)
	if _, err := os.Lstat(p); os.IsNotExist(err) {
		comm.Logf("No saved Steam credentials at %s", p)
		comm.Log("Nothing to do.")
		return nil
	}

	comm.Notice("Important note", []string{
		"This removes the Steam login and publisher key saved by butler.",
		"It does not revoke them on Steam's side. To do that, sign out of",
		"other devices in your Steam account settings, and regenerate the",
		"publisher key from your partner group page.",
	})
	comm.Logf("")
	if !comm.YesNo("Do you want to erase your saved Steam credentials?") {
		comm.Log("Okay, not erasing Steam credentials.")
		return nil
	}

	if err := os.Remove(p); err != nil {
		return errors.Wrap(err, "deleting steam credentials")
	}
	if err := os.Remove(keysPath(ctx)); err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "deleting cached depot keys")
	}
	comm.Log("Erased the Steam credentials saved on this computer.")
	return nil
}

func doKey(ctx *mansion.Context) {
	ctx.Must(Key(ctx, keyArgs.key))
}

func Key(ctx *mansion.Context, key string) error {
	goCtx, cancel := ctx.DefaultCtx()
	defer cancel()

	if key == "" {
		comm.Logf("Create a publisher Web API key at https://partner.steamgames.com/pub/groups/ under your publisher group.")
		var err error
		key, err = prompt("Publisher Web API key: ", true)
		if err != nil {
			return err
		}
	}
	if key == "" {
		return errors.New("no key given")
	}

	apps, err := partner.NewClient(key).Apps(goCtx)
	if err != nil {
		return errors.Wrap(err, "verifying publisher key")
	}

	c, err := loadCreds(ctx)
	if err != nil {
		return err
	}
	c.PublisherKey = key
	if err := saveCreds(ctx, c); err != nil {
		return err
	}
	comm.Logf("Key verified, it controls %d app(s). Saved to %s", len(apps), credsPath(ctx))
	comm.Result(map[string]interface{}{"status": "success", "app_count": len(apps)})
	return nil
}
