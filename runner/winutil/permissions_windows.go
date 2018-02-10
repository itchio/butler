// +build windows

package winutil

import (
	"syscall"
	"unsafe"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/runner/syscallex"
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

type SetFilePermissionsParams struct {
	FilePath         string
	Trustee          string
	PermissionChange PermissionChange

	// syscallex.GENERIC_READ etc.
	AccessRights uint32

	// syscallex.OBJECT_INHERIT_ACE etc.
	Inheritance uint32
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
		return errors.Wrap(err, 0)
	}
	defer SafeRelease(pSD)

	// Initialize an EXPLICIT_ACCESS structure for the new ACE
	var ea syscallex.ExplicitAccess
	ea.AccessPermissions = params.AccessRights
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
		return errors.Wrap(err, 0)
	}
	defer SafeRelease(uintptr(unsafe.Pointer(pNewDACL)))

	// Convert the security descriptor to absolute format
	var absoluteSDSize uint32
	var daclSize uint32
	var saclSize uint32
	var ownerSize uint32
	var groupSize uint32

	// sic. ignoring err on purpose
	syscallex.MakeAbsoluteSD(
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
		return errors.Wrap(err, 0)
	}

	// Attach the new ACL as the object's DACL
	err = syscallex.SetSecurityDescriptorDacl(
		uintptr(unsafe.Pointer(&pAbsoluteSD[0])),
		1, // bDaclPresent
		pNewDACL,
		0, // not defaulted
	)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	switch params.Inheritance {
	case InheritanceModeNone:
		// use legacy SetFileSecurity call, which doesn't propagaate
		err = syscallex.SetFileSecurity(
			objectName,
			syscallex.DACL_SECURITY_INFORMATION,
			uintptr(unsafe.Pointer(&pAbsoluteSD[0])),
		)
		if err != nil {
			return errors.Wrap(err, 0)
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
			return errors.Wrap(err, 0)
		}
	}

	return nil
}

func SafeRelease(handle uintptr) {
	if handle != 0 {
		syscall.Close(syscall.Handle(handle))
	}
}
