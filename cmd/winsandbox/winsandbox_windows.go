// +build windows

// This package implements a sandbox for Windows. It works by
// creating a less-privileged user, `itch-player-XXXXX`, which
// we hide from login and share a game's folder before we launch
// it (then unshare it immediately after).
//
// If you want to see/manage the user the sandbox creates,
// you can use "lusrmgr.msc" on Windows (works in Win+R)
package winsandbox

import (
	"fmt"
	"syscall"
	"time"

	"github.com/itchio/butler/runner/syscallex"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/runner/winutil"
	"github.com/itchio/wharf/state"

	"github.com/itchio/butler/mansion"
)

var setfilepermissionsArgs = struct {
	file    *string
	change  *string
	rights  *string
	trustee *string
	inherit *bool
}{}

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

	{
		cmd := parentCmd.Command("setfilepermissions", "Set up the sandbox (requires elevation)").Hidden()
		setfilepermissionsArgs.file = cmd.Arg("file", "Name of file (or directory) to manipulate").Required().String()
		setfilepermissionsArgs.change = cmd.Arg("change", "Operation").Required().Enum("grant", "revoke")
		setfilepermissionsArgs.rights = cmd.Arg("rights", "Rights to grant/revoke").Required().Enum("read", "all")
		setfilepermissionsArgs.trustee = cmd.Arg("trustee", "Name of trustee").Required().String()
		setfilepermissionsArgs.inherit = cmd.Flag("inherit", "Whether to inherit").Required().Bool()
		ctx.Register(cmd, doSetfilepermissions)
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

	consumer.Opf("Trying to log in...")

	token, err := winutil.Logon(pd.Username, ".", pd.Password)

	if err != nil {
		rescued := false

		if en, ok := winutil.AsErrno(err); ok {
			switch en {
			case syscallex.ERROR_PASSWORD_EXPIRED:
			case syscallex.ERROR_PASSWORD_MUST_CHANGE:
				// Some Windows versions (10 for example) expire password automatically.
				// Thankfully, we can renew it without administrator access, simply by using the old one.
				consumer.Opf("Password has expired, setting new password...")
				newPassword := generatePassword()

				err := syscallex.NetUserChangePassword(
					nil, // domainname
					syscall.StringToUTF16Ptr(pd.Username),
					syscall.StringToUTF16Ptr(pd.Password),
					syscall.StringToUTF16Ptr(newPassword),
				)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				pd.Password = newPassword
				err = pd.Save()
				if err != nil {
					return errors.Wrap(err, 0)
				}

				token, err = winutil.Logon(pd.Username, ".", pd.Password)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				consumer.Statf("Set new password successfully!")

				rescued = true
			}
		}

		if !rescued {
			return errors.Wrap(err, 0)
		}
	}
	defer syscall.CloseHandle(token)

	consumer.Statf("Everything looks good!")

	return nil
}

func doSetup(ctx *mansion.Context) {
	ctx.Must(Setup(comm.NewStateConsumer()))
}

func Setup(consumer *state.Consumer) error {
	startTime := time.Now()

	nullConsumer := &state.Consumer{}

	err := Check(nullConsumer)
	if err == nil {
		consumer.Statf("Already set up properly!")
		return nil
	}

	username := fmt.Sprintf("itch-player-%x", time.Now().Unix())
	comm.Opf("Generated username (%s)", username)

	password := generatePassword()
	comm.Opf("Generated password (%s)", password)

	comment := "itch.io sandbox user"

	comm.Opf("Adding user...")

	err = winutil.AddUser(username, password, comment)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Opf("Removing from Users group (so it doesn't show up as a login option)...")

	err = winutil.RemoveUserFromUsersGroup(username)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Opf("Loading profile for the first time (to create some directories)...")

	err = winutil.LoadProfileOnce(username, ".", password)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Opf("Saving to credentials registry...")

	pd := &PlayerData{
		Username: username,
		Password: password,
	}
	err = pd.Save()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Statf("All done! (in %s)", time.Since(startTime))

	return nil
}

func doSetfilepermissions(ctx *mansion.Context) {
	ctx.Must(Setfilepermissions(comm.NewStateConsumer()))
}

func Setfilepermissions(consumer *state.Consumer) error {
	params := &winutil.SetFilePermissionsParams{
		FilePath: *setfilepermissionsArgs.file,
		Trustee:  *setfilepermissionsArgs.trustee,
	}

	switch *setfilepermissionsArgs.change {
	case "grant":
		consumer.Opf("Granting...")
		params.PermissionChange = winutil.PermissionChangeGrant
	case "revoke":
		consumer.Opf("Revoking...")
		params.PermissionChange = winutil.PermissionChangeRevoke
	default:
		return fmt.Errorf("unknown change: %s", *setfilepermissionsArgs.change)
	}

	if *setfilepermissionsArgs.inherit {
		consumer.Opf("With inheritance...")
		params.Inheritance = winutil.InheritanceModeFull
	} else {
		consumer.Opf("Without inheritance...")
		params.Inheritance = winutil.InheritanceModeNone
	}

	switch *setfilepermissionsArgs.rights {
	case "read":
		consumer.Opf("Read access...")
		params.AccessRights = syscallex.GENERIC_READ
	case "all":
		consumer.Opf("All access...")
		params.AccessRights = syscallex.GENERIC_READ | syscallex.GENERIC_WRITE | syscallex.GENERIC_EXECUTE | syscallex.GENERIC_ALL
	default:
		return fmt.Errorf("unknown rights: %s", *setfilepermissionsArgs.rights)
	}

	consumer.Opf("To %s...", *setfilepermissionsArgs.trustee)
	consumer.Opf("For %s", *setfilepermissionsArgs.file)

	err := winutil.SetFilePermissions(params)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
