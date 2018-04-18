// +build windows

package winox

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"github.com/itchio/ox/syscallex"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type PermissionChange int

const (
	PermissionChangeGrant = iota
	PermissionChangeRevoke
)

type InheritanceMode int

const (
	InheritanceModeNone = iota
	InheritanceModeFull
)

type Rights uint32

const (
	RightsRead    = syscallex.GENERIC_READ
	RightsWrite   = syscallex.GENERIC_WRITE
	RightsExecute = syscallex.GENERIC_EXECUTE
	RightsAll     = syscallex.GENERIC_ALL

	RightsFull = RightsRead | RightsWrite | RightsExecute | RightsAll
)

type SetFilePermissionsParams struct {
	FilePath         string
	Trustee          string
	PermissionChange PermissionChange

	AccessRights Rights
	Inheritance  InheritanceMode
}

func SetFilePermissions(params *SetFilePermissionsParams) error {
	if params.FilePath == "" {
		return errors.New("FilePath cannot be empty")
	}
	if params.Trustee == "" {
		return errors.New("Trustee cannot be empty")
	}

	objectName := syscall.StringToUTF16Ptr(params.FilePath)
	var objectType uint32 = syscallex.SE_FILE_OBJECT

	var accessMode uint32
	switch params.PermissionChange {
	case PermissionChangeGrant:
		accessMode = syscallex.GRANT_ACCESS
	case PermissionChangeRevoke:
		accessMode = syscallex.REVOKE_ACCESS
	default:
		return errors.New("unknown PermissionChange value")
	}

	var inheritance uint32
	switch params.Inheritance {
	case InheritanceModeNone:
		inheritance = syscallex.NO_INHERITANCE
	case InheritanceModeFull:
		inheritance = syscallex.CONTAINER_INHERIT_ACE | syscallex.OBJECT_INHERIT_ACE
	default:
		return errors.New("unknown Inheritance value")
	}

	// Get a pointer to the existing DACL
	var pOldDACL *syscallex.ACL
	var pSD uintptr
	err := syscallex.GetNamedSecurityInfo(
		objectName,
		objectType,
		syscallex.DACL_SECURITY_INFORMATION,
		0,         // ppsidOwner
		0,         // ppsidGroup
		&pOldDACL, // ppDacl
		nil,       // ppSacl
		uintptr(unsafe.Pointer(&pSD)), // ppSecurityDescriptor
	)
	if err != nil {
		return errors.WithStack(err)
	}
	defer SafeRelease(pSD)

	// Initialize an EXPLICIT_ACCESS structure for the new ACE
	var ea syscallex.ExplicitAccess
	ea.AccessPermissions = uint32(params.AccessRights)
	ea.AccessMode = accessMode
	ea.Inheritance = inheritance
	ea.Trustee.TrusteeForm = syscallex.TRUSTEE_IS_NAME
	ea.Trustee.Name = syscall.StringToUTF16Ptr(params.Trustee)

	// Create a new ACL that merges the new ACE
	// into the existing DACL.
	var pNewDACL *syscallex.ACL
	err = syscallex.SetEntriesInAcl(
		1, // number of items
		uintptr(unsafe.Pointer(&ea)), // pointer to first element of array
		pOldDACL,
		&pNewDACL,
	)
	if err != nil {
		return errors.WithStack(err)
	}
	defer SafeRelease(uintptr(unsafe.Pointer(pNewDACL)))

	switch params.Inheritance {
	case InheritanceModeNone:
		// use legacy SetFileSecurity call, which doesn't propagaate

		// But first, convert the (self-relative) security descriptor to absolute format
		var absoluteSDSize uint32
		var daclSize uint32
		var saclSize uint32
		var ownerSize uint32
		var groupSize uint32

		// sic. ignoring err on purpose
		err = syscallex.MakeAbsoluteSD(
			pSD,
			0,
			&absoluteSDSize,
			nil,
			&daclSize,
			nil,
			&saclSize,
			0,
			&ownerSize,
			0,
			&groupSize,
		)
		if err != nil {
			rescued := false
			if en, ok := AsErrno(err); ok {
				if en == syscall.ERROR_INSUFFICIENT_BUFFER {
					// cool, that's expected!
					rescued = true
				}
			}

			if !rescued {
				return errors.WithStack(err)
			}
		}

		// allocate everything
		// avoid 0-length allocations because then the
		// uintptr(unsafe.Pointer(&slice[0])) doesn't work
		pAbsoluteSD := make([]byte, absoluteSDSize+1)
		pDacl := make([]byte, daclSize+1)
		pSacl := make([]byte, saclSize+1)
		pOwner := make([]byte, ownerSize+1)
		pGroup := make([]byte, groupSize+1)

		err = syscallex.MakeAbsoluteSD(
			pSD,
			uintptr(unsafe.Pointer(&pAbsoluteSD[0])),
			&absoluteSDSize,
			(*syscallex.ACL)(unsafe.Pointer(&pDacl[0])),
			&daclSize,
			(*syscallex.ACL)(unsafe.Pointer(&pSacl[0])),
			&saclSize,
			uintptr(unsafe.Pointer(&pOwner[0])),
			&ownerSize,
			uintptr(unsafe.Pointer(&pGroup[0])),
			&groupSize,
		)
		if err != nil {
			return errors.WithStack(err)
		}

		// Attach the new ACL as the object's DACL
		err = syscallex.SetSecurityDescriptorDacl(
			uintptr(unsafe.Pointer(&pAbsoluteSD[0])),
			1, // bDaclPresent
			pNewDACL,
			0, // not defaulted
		)
		if err != nil {
			return errors.WithStack(err)
		}
		err = syscallex.SetFileSecurity(
			objectName,
			syscallex.DACL_SECURITY_INFORMATION,
			uintptr(unsafe.Pointer(&pAbsoluteSD[0])),
		)
		if err != nil {
			return errors.WithStack(err)
		}
	case InheritanceModeFull:
		// use new SetNamedSecurityInfo call, which propagates
		err = syscallex.SetNamedSecurityInfo(
			objectName,
			objectType,
			syscallex.DACL_SECURITY_INFORMATION,
			0,        // psidOwner
			0,        // psidGroup
			pNewDACL, // pDacl
			nil,      // pSacl
		)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func SafeRelease(handle uintptr) {
	if handle != 0 {
		syscall.Close(syscall.Handle(handle))
	}
}

type ShareEntry struct {
	Path        string
	Inheritance InheritanceMode
	Rights      Rights
}

func (se *ShareEntry) Grant(trustee string) error {
	err := SetFilePermissions(se.params(PermissionChangeGrant, trustee))
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (se *ShareEntry) Revoke(trustee string) error {
	err := SetFilePermissions(se.params(PermissionChangeRevoke, trustee))
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (se *ShareEntry) params(change PermissionChange, trustee string) *SetFilePermissionsParams {
	return &SetFilePermissionsParams{
		FilePath:         se.Path,
		AccessRights:     se.Rights,
		PermissionChange: change,
		Trustee:          trustee,
		Inheritance:      se.Inheritance,
	}
}

type SharingPolicy struct {
	Trustee string
	Entries []*ShareEntry
}

func (sp *SharingPolicy) Grant(consumer *state.Consumer) error {
	ec := &errorCoalescer{
		operation: "granting permissions",
		consumer:  consumer,
	}
	for _, se := range sp.Entries {
		ec.Record(se.Grant(sp.Trustee))
	}
	return ec.Result()
}

func (sp *SharingPolicy) Revoke(consumer *state.Consumer) error {
	ec := &errorCoalescer{
		operation: "revoking permissions",
		consumer:  consumer,
	}
	for _, se := range sp.Entries {
		ec.Record(se.Revoke(sp.Trustee))
	}
	return ec.Result()
}

func (sp *SharingPolicy) String() string {
	var entries []string

	for _, e := range sp.Entries {
		perms := ""
		if e.Rights&RightsRead > 0 {
			perms += "R"
		}
		if e.Rights&RightsWrite > 0 {
			perms += "W"
		}
		if e.Rights&RightsExecute > 0 {
			perms += "X"
		}
		if e.Rights&RightsAll > 0 {
			perms += "*"
		}

		inherit := ""
		if e.Inheritance == InheritanceModeFull {
			inherit = "(CI)(OI)"
		} else {
			inherit = ""
		}

		entries = append(entries, fmt.Sprintf("  → (%s)(%s)%s", e.Path, perms, inherit))
	}

	var entriesString = "  (no sharing entries)"
	if len(entries) > 0 {
		entriesString = strings.Join(entries, "\n")
	}

	return fmt.Sprintf("for %s\n%s", sp.Trustee, entriesString)
}

type errorCoalescer struct {
	operation string
	consumer  *state.Consumer

	// internal
	errors []error
}

func (ec *errorCoalescer) Record(err error) {
	if err != nil {
		ec.errors = append(ec.errors, err)
		ec.consumer.Warnf("While %s: %+v", ec.operation, err)
	}
}

func (ec *errorCoalescer) Result() error {
	if len(ec.errors) > 0 {
		var messages []string
		for _, e := range ec.errors {
			messages = append(messages, e.Error())
		}
		return fmt.Errorf("%d errors while %s: %s", len(messages), ec.operation, strings.Join(messages, " ; "))
	}
	return nil
}

func GetImpersonationToken(username string, domain string, password string) (syscall.Token, error) {
	var impersonationToken syscall.Token
	err := Impersonate(username, domain, password, func() error {
		currentThread := syscallex.GetCurrentThread()

		err := syscallex.OpenThreadToken(
			currentThread,
			syscall.TOKEN_ALL_ACCESS,
			1,
			&impersonationToken,
		)
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return impersonationToken, nil
}

func UserHasPermission(impersonationToken syscall.Token, accessDesired uint32, path string) (bool, error) {
	// cf. http://blog.aaronballman.com/2011/08/how-to-check-access-rights/
	// (more or less)

	// get the security descriptor for the file
	var securityDescriptorLength uint32
	syscallex.GetFileSecurity(
		syscall.StringToUTF16Ptr(path),
		syscallex.OWNER_SECURITY_INFORMATION|syscallex.GROUP_SECURITY_INFORMATION|syscallex.DACL_SECURITY_INFORMATION,
		0,
		0,
		&securityDescriptorLength,
	)

	// allow 0-length allocations
	securityDescriptor := make([]byte, securityDescriptorLength+1)
	err := syscallex.GetFileSecurity(
		syscall.StringToUTF16Ptr(path),
		syscallex.OWNER_SECURITY_INFORMATION|syscallex.GROUP_SECURITY_INFORMATION|syscallex.DACL_SECURITY_INFORMATION,
		uintptr(unsafe.Pointer(&securityDescriptor[0])),
		securityDescriptorLength,
		&securityDescriptorLength,
	)
	if err != nil {
		return false, errors.WithStack(err)
	}

	var accessStatus bool

	var mapping syscallex.GenericMapping
	mapping.GenericRead = syscallex.FILE_GENERIC_READ
	mapping.GenericWrite = syscallex.FILE_GENERIC_WRITE
	mapping.GenericExecute = syscallex.FILE_GENERIC_EXECUTE
	mapping.GenericAll = syscallex.FILE_ALL_ACCESS
	syscallex.MapGenericMask(&accessDesired, &mapping)

	var grantedAccess uint32
	var privilegeSetLength uint32

	// get length of privilegeSet
	syscallex.AccessCheck(
		uintptr(unsafe.Pointer(&securityDescriptor[0])),
		impersonationToken,
		accessDesired,
		&mapping,
		0,
		&privilegeSetLength,
		&grantedAccess,
		&accessStatus,
	)

	// avoid 0-byte allocation
	privilegeSet := make([]byte, privilegeSetLength+1)

	err = syscallex.AccessCheck(
		uintptr(unsafe.Pointer(&securityDescriptor[0])),
		impersonationToken,
		accessDesired,
		&mapping,
		uintptr(unsafe.Pointer(&privilegeSet[0])),
		&privilegeSetLength,
		&grantedAccess,
		&accessStatus,
	)
	if err != nil {
		return false, errors.WithStack(err)
	}

	return accessStatus, nil
}
