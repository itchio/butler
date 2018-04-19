//+build windows

package fuji

import (
	"fmt"

	"github.com/itchio/ox/syscallex"

	"github.com/itchio/ox/winox"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type SetFilePermissionParams struct {
	// The file or folder to apply permissions to
	Path string
	// Inherit determines whether permissions also apply to all folders and files
	// in the container, recursively
	Inherit bool
	// Rights if one of "read", "write", "execute", "all", or "full"
	Rights string
	// Trustee is the name of the account permissions are granted to or revoked
	Trustee string
	// Change is one of "grant" or "revoke"
	Change string
}

func SetFilePermissions(consumer *state.Consumer, params *SetFilePermissionParams) error {
	entry := &winox.ShareEntry{
		Path: params.Path,
	}

	if params.Inherit {
		entry.Inheritance = winox.InheritanceModeFull
	} else {
		entry.Inheritance = winox.InheritanceModeNone
	}

	switch params.Rights {
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
		return fmt.Errorf("unknown rights: %s", params.Rights)
	}

	policy := &winox.SharingPolicy{
		Trustee: params.Trustee,
		Entries: []*winox.ShareEntry{entry},
	}

	switch params.Change {
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
		return fmt.Errorf("unknown change: %s", params.Change)
	}

	consumer.Statf("Policy applied successfully")

	return nil
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

type CheckAccessParams struct {
	File string
}

func (i *instance) CheckAccess(consumer *state.Consumer, params *CheckAccessParams) error {
	creds, err := i.GetCredentials()
	if err != nil {
		return errors.WithStack(err)
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
			params.File,
		)
		if err != nil {
			return errors.WithStack(err)
		}

		if hasAccess {
			consumer.Opf("User has %s access", spec.name)
		} else {
			consumer.Opf("User does not have %s access", spec.name)
		}
	}

	return nil
}
