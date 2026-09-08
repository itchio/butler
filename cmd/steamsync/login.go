package steamsync

import (
	"context"
	"fmt"
	"os"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/butler/steam"
	"github.com/itchio/fresh-steamer/auth"
	"github.com/mdp/qrterminal/v3"
	"github.com/pkg/errors"
)

var loginArgs = struct {
	password bool
	user     string
	noSave   bool
}{}

var keyArgs = struct {
	key string
}{}

func RegisterLogin(ctx *mansion.Context) {
	cmd := ctx.App.Command("steam-login", "Log in to a Steam account so butler can download your builds.").Hidden()
	cmd.Flag("password", "Log in with account name and password instead of scanning a QR code").BoolVar(&loginArgs.password)
	cmd.Flag("user", "Steam account name (password login only)").StringVar(&loginArgs.user)
	cmd.Flag("no-save", "Print the login token instead of storing it, for passing to later commands via --steam-refresh-token or "+steam.EnvRefreshToken+".").BoolVar(&loginArgs.noSave)
	ctx.Register(cmd, doLogin)

	logout := ctx.App.Command("steam-logout", "Remove saved Steam credentials and publisher key.").Hidden()
	ctx.Register(logout, doLogout)

	key := ctx.App.Command("steam-key", "Store a Steam publisher Web API key. It proves which apps you control.").Hidden()
	key.Arg("key", "The key. Prompted for when omitted.").StringVar(&keyArgs.key)
	ctx.Register(key, doKey)
}

func doLogin(ctx *mansion.Context) {
	ctx.Must(Login(ctx, loginArgs.password, loginArgs.user, loginArgs.noSave))
}

func Login(ctx *mansion.Context, usePassword bool, user string, noSave bool) error {
	goCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := store(ctx)
	existing, err := st.Persisted()
	if err != nil {
		return err
	}
	if existing.LoggedIn() && !noSave {
		comm.Logf("Already logged in to Steam as %s. Run `butler steam-logout` to switch accounts.", existing.AccountName)
		comm.Result(map[string]string{"status": "success", "account_name": existing.AccountName})
		return nil
	}

	opts := steam.LoginOptions{Persist: !noSave}
	var login *steam.Creds
	if usePassword {
		login, err = loginPassword(goCtx, st, user, opts)
	} else {
		login, err = steam.LoginQR(goCtx, st, showChallenge, opts)
	}
	if err != nil {
		return err
	}

	if noSave {
		comm.Logf("Logged in to Steam as %s. Nothing was saved; the token below is the only copy.", login.AccountName)
		comm.ResultOrPrint(map[string]string{"status": "success", "account_name": login.AccountName, "refresh_token": login.RefreshToken}, func() {
			fmt.Println(login.RefreshToken)
		})
		return nil
	}

	comm.Logf("Logged in to Steam as %s, credentials saved to %s", login.AccountName, st.CredsPath())
	if !existing.HasPublisherKey() {
		comm.Logf("Next, run `butler steam-key` to store the publisher key that proves which apps you control.")
	}
	comm.Result(map[string]string{"status": "success", "account_name": login.AccountName})
	return nil
}

func showChallenge(url string) {
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
}

func loginPassword(goCtx context.Context, st steam.Store, name string, opts steam.LoginOptions) (*steam.Creds, error) {
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
	guard := auth.GuardFunc(func(ctx context.Context, kind auth.GuardType, msg string) (string, error) {
		if kind == auth.GuardDeviceConfirmation {
			comm.Logf("Approve the login in the Steam mobile app...")
			return "", nil
		}
		label := fmt.Sprintf("Steam Guard %s", kind)
		if msg != "" {
			label += " (" + msg + ")"
		}
		return prompt(label+": ", false)
	})
	return steam.LoginPassword(goCtx, st, name, pass, guard, opts)
}

func doLogout(ctx *mansion.Context) {
	ctx.Must(Logout(ctx))
}

func Logout(ctx *mansion.Context) error {
	st := store(ctx)
	if _, err := os.Lstat(st.CredsPath()); os.IsNotExist(err) {
		comm.Logf("No saved Steam credentials at %s", st.CredsPath())
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

	if err := st.Logout(); err != nil {
		return errors.Wrap(err, "deleting steam credentials")
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

	st := store(ctx)
	warnUngated()
	apps, err := steam.SetPublisherKey(goCtx, st, key)
	if err != nil {
		return err
	}
	if steam.Ungated() {
		comm.Logf("Key saved without verification to %s", st.CredsPath())
	} else {
		comm.Logf("Key verified, it controls %d app(s). Saved to %s", len(apps), st.CredsPath())
	}
	comm.Result(map[string]interface{}{"status": "success", "app_count": len(apps)})
	return nil
}
