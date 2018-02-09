// +build windows

package winsandbox

import (
	"fmt"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/runner/winutil"
	"github.com/itchio/wharf/state"

	"github.com/itchio/butler/mansion"
)

func Register(ctx *mansion.Context) {
	parentCmd := ctx.App.Command("winsandbox", "Use or manage the itch.io sandbox for Windows")

	{
		cmd := parentCmd.Command("check", "Verify that the sandbox is properly set up").Hidden()
		ctx.Register(cmd, doCheck)
	}

	{
		cmd := parentCmd.Command("setup", "Set up the sandbox (requires elevation)").Hidden()
		ctx.Register(cmd, doSetup)
	}
}

func doCheck(ctx *mansion.Context) {
	ctx.Must(Check(comm.NewStateConsumer()))
}

func Check(consumer *state.Consumer) error {
	consumer.Opf("Retrieving player data from registry...")
	pd, err := GetPlayerData()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Statf("Sandbox user is (%s)", pd.Username)

	consumer.Opf("Trying to impersonate")
	err = winutil.Impersonate(pd.Username, ".", pd.Password, func() error {
		return nil
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Statf("Everything looks good!")

	return nil
}

func doSetup(ctx *mansion.Context) {
	ctx.Must(Setup(comm.NewStateConsumer()))
}

func Setup(consumer *state.Consumer) error {
	nullConsumer := &state.Consumer{}

	err := Check(nullConsumer)
	if err == nil {
		consumer.Opf("Nothing to do for setup")
		return nil
	}

	username := fmt.Sprintf("itch-player-%x", time.Now().Unix())
	comm.Opf("Generated username (%s)", username)

	password := generatePassword()
	comm.Opf("Generated password (%s)", password)

	comment := "itch.io sandbox user"

	err = winutil.AddUser(username, password, comment)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return errors.New("stub!")
}
