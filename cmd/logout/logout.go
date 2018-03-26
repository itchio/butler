package logout

import (
	"fmt"
	"os"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/pkg/errors"
)

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("logout", "Remove saved itch.io credentials.")
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx))
}

func Do(ctx *mansion.Context) error {
	var identity = ctx.Identity

	_, err := os.Lstat(identity)
	if err != nil {
		if os.IsNotExist(err) {
			comm.Logf("No saved credentials at %s", identity)
			comm.Log("Nothing to do.")
			return nil
		}
	}

	comm.Notice("Important note", []string{
		"Note: this command will not invalidate the API key itself.",
		"If you wish to revoke it (for example, because it's been compromised), you should do so in your user settings:",
		"",
		fmt.Sprintf("  %s/user/settings\n\n", ctx.Address),
	})

	comm.Logf("")

	if !comm.YesNo("Do you want to erase your saved API key?") {
		comm.Log("Okay, not erasing credentials. Bye!")
		return nil
	}

	err = os.Remove(identity)
	if err != nil {
		return errors.Wrap(err, "deleting identity file")
	}

	comm.Log("You've successfully erased the API key that was saved on your computer.")

	return nil
}
