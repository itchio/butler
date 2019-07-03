// +build windows

// This package implements a sandbox for Windows. It works by
// creating a less-privileged user, `itch-player-XXXXX`, which
// we hide from login and share a game's folder before we launch
// it (then unshare it immediately after).
//
// If you want to see/manage the user the sandbox creates,
// you can use "lusrmgr.msc" on Windows (works in Win+R)
package fujicmd

import (
	"fmt"

	"github.com/itchio/smaug/fuji"

	"github.com/itchio/ox/syscallex"

	"github.com/itchio/butler/comm"
	"github.com/itchio/ox/winox"
	"github.com/itchio/headway/state"
	"github.com/pkg/errors"

	"github.com/itchio/butler/mansion"
)

var setfilepermissionsArgs = struct {
	file    *string
	change  *string
	rights  *string
	trustee *string
	inherit *bool
}{}

var checkAccessArgs = struct {
	file *string
}{}

func Register(ctx *mansion.Context) {
	parentCmd := ctx.App.Command("fuji", "Use or manage the itch.io sandbox for Windows").Hidden()

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
		setfilepermissionsArgs.rights = cmd.Arg("rights", "Rights to grant/revoke").Required().Enum("read", "write", "execute", "all", "full")
		setfilepermissionsArgs.trustee = cmd.Arg("trustee", "Name of trustee").Required().String()
		setfilepermissionsArgs.inherit = cmd.Flag("inherit", "Whether to inherit").Required().Bool()
		ctx.Register(cmd, doSetfilepermissions)
	}

	{
		cmd := parentCmd.Command("checkaccess", "Check if the sandbox user has access to a certain file").Hidden()
		checkAccessArgs.file = cmd.Arg("file", "Name of file (or directory) to check access for").Required().String()
		ctx.Register(cmd, doCheckAccess)
	}
}

func doCheck(ctx *mansion.Context) {
	ctx.Must(Check(comm.NewStateConsumer()))
}

func Check(consumer *state.Consumer) error {
	i, err := fuji.NewInstance(mansion.GetFujiSettings())
	if err != nil {
		return err
	}

	return i.Check(consumer)
}

func doSetup(ctx *mansion.Context) {
	ctx.Must(Setup(comm.NewStateConsumer()))
}

func Setup(consumer *state.Consumer) error {
	i, err := fuji.NewInstance(mansion.GetFujiSettings())
	if err != nil {
		return err
	}

	return i.Setup(consumer)
}

func doSetfilepermissions(ctx *mansion.Context) {
	ctx.Must(Setfilepermissions(comm.NewStateConsumer()))
}

func Setfilepermissions(consumer *state.Consumer) error {
	entry := &winox.ShareEntry{
		Path: *setfilepermissionsArgs.file,
	}

	if *setfilepermissionsArgs.inherit {
		entry.Inheritance = winox.InheritanceModeFull
	} else {
		entry.Inheritance = winox.InheritanceModeNone
	}

	switch *setfilepermissionsArgs.rights {
	case "read":
		entry.Rights = winox.RightsRead
	case "write":
		entry.Rights = winox.RightsWrite
	case "execute":
		entry.Rights = winox.RightsExecute
	case "all":
		entry.Rights = winox.RightsAll
	case "full":
		entry.Rights = winox.RightsFull
	default:
		return fmt.Errorf("unknown rights: %s", *setfilepermissionsArgs.rights)
	}

	policy := &winox.SharingPolicy{
		Trustee: *setfilepermissionsArgs.trustee,
		Entries: []*winox.ShareEntry{entry},
	}

	switch *setfilepermissionsArgs.change {
	case "grant":
		consumer.Opf("Granting %s", policy)
		err := policy.Grant(consumer)
		if err != nil {
			return errors.WithStack(err)
		}
	case "revoke":
		consumer.Opf("Revoking %s", policy)
		err := policy.Revoke(consumer)
		if err != nil {
			return errors.WithStack(err)
		}
	default:
		return fmt.Errorf("unknown change: %s", *setfilepermissionsArgs.change)
	}

	comm.Statf("Policy applied successfully")

	return nil
}

func doCheckAccess(ctx *mansion.Context) {
	ctx.Must(CheckAccess(comm.NewStateConsumer()))
}

type checkAccessSpec struct {
	name  string
	flags uint32
}

var checkAccessSpecs = []checkAccessSpec{
	{"read", syscallex.GENERIC_READ},
	{"write", syscallex.GENERIC_WRITE},
	{"execute", syscallex.GENERIC_EXECUTE},
	{"all", syscallex.GENERIC_ALL},
}

func CheckAccess(consumer *state.Consumer) error {
	i, err := fuji.NewInstance(mansion.GetFujiSettings())
	if err != nil {
		return err
	}

	creds, err := i.GetCredentials()
	if err != nil {
		return err
	}

	impersonationToken, err := winox.GetImpersonationToken(creds.Username, ".", creds.Password)
	if err != nil {
		return errors.WithStack(err)
	}
	defer winox.SafeRelease(uintptr(impersonationToken))

	for _, spec := range checkAccessSpecs {
		hasAccess, err := winox.UserHasPermission(
			impersonationToken,
			spec.flags,
			*checkAccessArgs.file,
		)
		if err != nil {
			return errors.WithStack(err)
		}

		if hasAccess {
			comm.Opf("User has %s access", spec.name)
		} else {
			comm.Opf("User does not have %s access", spec.name)
		}
	}

	return nil
}
