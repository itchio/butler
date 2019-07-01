package login

import (
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/pkg/errors"
)

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("login", "Connect butler to your itch.io account and save credentials locally.")
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx))
}

func Do(ctx *mansion.Context) error {
	if ctx.HasSavedCredentials() {
		client, err := ctx.AuthenticateViaOauth()
		if err != nil {
			return errors.Wrap(err, "authenticating with saved credentials")
		}

		_, err = client.WharfStatus(ctx.DefaultCtx())
		if err != nil {
			return errors.Wrap(err, "validating credentials")
		}

		comm.Logf("Your local credentials are valid!\n")
		comm.Logf("If you want to log in as another account, use the `butler logout` command first, or specify a different credentials path with the `-i` flag.")
		comm.Result(map[string]string{"status": "success"})
	} else {
		// this does the full login flow + saves
		_, err := ctx.AuthenticateViaOauth()
		if err != nil {
			return errors.Wrap(err, "authenticating (no saved credentials)")
		}
		comm.Result(map[string]string{"status": "success"})
		return nil
	}

	return nil
}
