package syscallex

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// logon flags
const (
	LOGON_WITH_PROFILE     = 1
	LOGON_CREDENTIALS_ONLY = 2
)

// logon type
const (
	LOGON32_LOGON_INTERACTIVE = 2
)

// logon provider
const (
	LOGON32_PROVIDER_DEFAULT = 0
)

// security impersonation level
const (
	SecurityAnonymous = iota
	SecurityIdentification
	SecurityImpersonation
	SecurityDelegation
)

// token types
const (
	TokenPrimary       = 1
	TokenImpersonation = 2
)

var (
	modadvapi32 = windows.NewLazySystemDLL("advapi32.dll")

	procCreateProcessWithLogonW = modadvapi32.NewProc("CreateProcessWithLogonW")
	procLogonUserW              = modadvapi32.NewProc("LogonUserW")
	procImpersonateLoggedOnUser = modadvapi32.NewProc("ImpersonateLoggedOnUser")
	procRevertToSelf            = modadvapi32.NewProc("RevertToSelf")
	procLookupAccountNameW      = modadvapi32.NewProc("LookupAccountNameW")
	procLookupAccountSidW       = modadvapi32.NewProc("LookupAccountSidW")
	procCreateWellKnownSid      = modadvapi32.NewProc("CreateWellKnownSid")

	procGetNamedSecurityInfoW     = modadvapi32.NewProc("GetNamedSecurityInfoW")
	procSetNamedSecurityInfoW     = modadvapi32.NewProc("SetNamedSecurityInfoW")
	procSetEntriesInAclW          = modadvapi32.NewProc("SetEntriesInAclW")
	procMakeAbsoluteSD            = modadvapi32.NewProc("MakeAbsoluteSD")
	procSetSecurityDescriptorDacl = modadvapi32.NewProc("SetSecurityDescriptorDacl")
	procGetFileSecurityW          = modadvapi32.NewProc("GetFileSecurityW")
	procSetFileSecurityW          = modadvapi32.NewProc("SetFileSecurityW")
	procAccessCheck               = modadvapi32.NewProc("AccessCheck")
	procMapGenericMask            = modadvapi32.NewProc("MapGenericMask")
)

func CreateProcessWithLogon(
	username *uint16,
	domain *uint16,
	password *uint16,
	logonFlags uint32,
	appName *uint16,
	commandLine *uint16,
	creationFlags uint32,
	env *uint16,
	currentDir *uint16,
	startupInfo *syscall.StartupInfo,
	outProcInfo *syscall.ProcessInformation,
) (err error) {
	r1, _, e1 := syscall.Syscall12(
		procCreateProcessWithLogonW.Addr(),
		11,
		uintptr(unsafe.Pointer(username)),
		uintptr(unsafe.Pointer(domain)),
		uintptr(unsafe.Pointer(password)),
		uintptr(logonFlags),
		uintptr(unsafe.Pointer(appName)),
		uintptr(unsafe.Pointer(commandLine)),
		uintptr(creationFlags),
		uintptr(unsafe.Pointer(env)),
		uintptr(unsafe.Pointer(currentDir)),
		uintptr(unsafe.Pointer(startupInfo)),
		uintptr(unsafe.Pointer(outProcInfo)),
		0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func LogonUser(
	username *uint16,
	domain *uint16,
	password *uint16,
	logonType uint32,
	logonProvider uint32,
	outToken *syscall.Token,
) (err error) {
	r1, _, e1 := syscall.Syscall6(
		procLogonUserW.Addr(),
		6,
		uintptr(unsafe.Pointer(username)),
		uintptr(unsafe.Pointer(domain)),
		uintptr(unsafe.Pointer(password)),
		uintptr(logonType),
		uintptr(logonProvider),
		uintptr(unsafe.Pointer(outToken)),
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func ImpersonateLoggedOnUser(
	token syscall.Token,
) (err error) {
	r1, _, e1 := syscall.Syscall(
		procImpersonateLoggedOnUser.Addr(),
		1,
		uintptr(token),
		0, 0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func RevertToSelf() (err error) {
	r1, _, e1 := syscall.Syscall(
		procRevertToSelf.Addr(),
		0,
		0, 0, 0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func LookupAccountName(
	systemName *uint16,
	accountName *uint16,
	sid uintptr,
	cbSid *uint32,
	referencedDomainName *uint16,
	cchReferencedDomainName *uint32,
	use *uint32,
) (err error) {
	r1, _, e1 := syscall.Syscall9(
		procLookupAccountNameW.Addr(),
		7,
		uintptr(unsafe.Pointer(systemName)),
		uintptr(unsafe.Pointer(accountName)),
		sid,
		uintptr(unsafe.Pointer(cbSid)),
		uintptr(unsafe.Pointer(referencedDomainName)),
		uintptr(unsafe.Pointer(cchReferencedDomainName)),
		uintptr(unsafe.Pointer(use)),
		0, 0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func LookupAccountSid(
	systemName *uint16,
	sid uintptr,
	name *uint16,
	cchName *uint32,
	referencedDomainName *uint16,
	cchReferencedDomainName *uint32,
	use *uint32,
) (err error) {
	r1, _, e1 := syscall.Syscall9(
		procLookupAccountSidW.Addr(),
		7,
		uintptr(unsafe.Pointer(systemName)),
		sid,
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(cchName)),
		uintptr(unsafe.Pointer(referencedDomainName)),
		uintptr(unsafe.Pointer(cchReferencedDomainName)),
		uintptr(unsafe.Pointer(use)),
		0, 0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func CreateWellKnownSid(
	wellKnownSidType int,
	domainSid uintptr,
	sid uintptr,
	cbSid *uint32,
) (err error) {
	r1, _, e1 := syscall.Syscall6(
		procCreateWellKnownSid.Addr(),
		4,
		uintptr(wellKnownSidType),
		domainSid,
		sid,
		uintptr(unsafe.Pointer(cbSid)),
		0, 0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

// SE_OBJECT_TYPE, cf.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa379593(v=vs.85).aspx
// do not reorder
const (
	SE_UNKNOWN_OBJECT_TYPE = iota
	SE_FILE_OBJECT
	SE_SERVICE
	SE_PRINTER
	SE_REGISTRY_KEY
	SE_LMSHARE
	SE_KERNEL_OBJECT
	SE_WINDOW_OBJECT
	SE_DS_OBJECT
	SE_DS_OBJECT_ALL
	SE_PROVIDER_DEFINED_OBJECT
	SE_WMIGUID_OBJECT
	SE_REGISTRY_WOW64_32KEY
)

// see
// https://raw.githubusercontent.com/mirror/reactos/master/reactos/include/xdk/setypes.h
const (
	DELETE                   = 0x00010000
	READ_CONTROL             = 0x00020000
	WRITE_DAC                = 0x00040000
	WRITE_OWNER              = 0x00080000
	SYNCHRONIZE              = 0x00100000
	STANDARD_RIGHTS_REQUIRED = 0x000F0000
	STANDARD_RIGHTS_READ     = READ_CONTROL
	STANDARD_RIGHTS_WRITE    = READ_CONTROL
	STANDARD_RIGHTS_EXECUTE  = READ_CONTROL
	STANDARD_RIGHTS_ALL      = 0x001F0000
	SPECIFIC_RIGHTS_ALL      = 0x0000FFFF
	ACCESS_SYSTEM_SECURITY   = 0x01000000
	MAXIMUM_ALLOWED          = 0x02000000
	GENERIC_READ             = 0x80000000
	GENERIC_WRITE            = 0x40000000
	GENERIC_EXECUTE          = 0x20000000
	GENERIC_ALL              = 0x10000000

	// cf. https://www.codeproject.com/script/Content/ViewAssociatedFile.aspx?rzp=%2FKB%2Fasp%2Fuseraccesscheck%2Fuseraccesscheck_demo.zip&zep=ASPDev%2FMasks.txt&obid=1881&obtid=2&ovid=1
	FILE_READ_DATA      = (0x0001) // file & pipe
	FILE_LIST_DIRECTORY = (0x0001) // directory

	FILE_WRITE_DATA = (0x0002) // file & pipe
	FILE_ADD_FILE   = (0x0002) // directory

	FILE_APPEND_DATA          = (0x0004) // file
	FILE_ADD_SUBDIRECTORY     = (0x0004) // directory
	FILE_CREATE_PIPE_INSTANCE = (0x0004) // named pipe

	FILE_READ_EA = (0x0008) // file & directory

	FILE_WRITE_EA = (0x0010) // file & directory

	FILE_EXECUTE  = (0x0020) // file
	FILE_TRAVERSE = (0x0020) // directory

	FILE_DELETE_CHILD = (0x0040) // directory

	FILE_READ_ATTRIBUTES = (0x0080) // all

	FILE_WRITE_ATTRIBUTES = (0x0100) // all

	FILE_ALL_ACCESS = (STANDARD_RIGHTS_REQUIRED | SYNCHRONIZE | 0x1FF)

	FILE_GENERIC_READ    = (STANDARD_RIGHTS_READ | FILE_READ_DATA | FILE_READ_ATTRIBUTES | FILE_READ_EA | SYNCHRONIZE)
	FILE_GENERIC_WRITE   = (STANDARD_RIGHTS_WRITE | FILE_WRITE_DATA | FILE_WRITE_ATTRIBUTES | FILE_WRITE_EA | FILE_APPEND_DATA | SYNCHRONIZE)
	FILE_GENERIC_EXECUTE = (STANDARD_RIGHTS_EXECUTE | FILE_READ_ATTRIBUTES | FILE_EXECUTE | SYNCHRONIZE)
)

// ACCESS_MODE, cf. https://msdn.microsoft.com/en-us/library/windows/desktop/aa374899(v=vs.85).aspx
// do not reorder
const (
	NOT_USED_ACCESS = iota
	GRANT_ACCESS
	SET_ACCESS
	DENY_ACCESS
	REVOKE_ACCESS
	SET_AUDIT_SUCCESS
	SET_AUDIT_FAILURE
)

// SECURITY_INFORMATION, cf.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa379573(v=vs.85).aspx
// and
// https://raw.githubusercontent.com/mirror/reactos/master/reactos/include/xdk/setypes.h
const (
	OWNER_SECURITY_INFORMATION = 0x00000001
	GROUP_SECURITY_INFORMATION = 0x00000002
	DACL_SECURITY_INFORMATION  = 0x00000004
	SACL_SECURITY_INFORMATION  = 0x00000008
	LABEL_SECURITY_INFORMATION = 0x00000010

	PROTECTED_DACL_SECURITY_INFORMATION   = 0x80000000
	PROTECTED_SACL_SECURITY_INFORMATION   = 0x40000000
	UNPROTECTED_DACL_SECURITY_INFORMATION = 0x20000000
	UNPROTECTED_SACL_SECURITY_INFORMATION = 0x10000000
)

// struct _ACL, cf.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa374931(v=vs.85).aspx
type ACL struct {
	AclRevision byte
	Sbz1        byte
	AclSize     int16
	AceCount    int16
	Sbz2        int16
}

func GetNamedSecurityInfo(
	objectName *uint16,
	objectType uint32,
	securityInfo uint32,
	ppsidOwner uintptr,
	ppsidGroup uintptr,
	ppDacl **ACL,
	ppSacl **ACL,
	ppSecurityDescriptor uintptr,
) (err error) {
	r1, _, _ := syscall.Syscall9(
		procGetNamedSecurityInfoW.Addr(),
		8,
		uintptr(unsafe.Pointer(objectName)),
		uintptr(objectType),
		uintptr(securityInfo),
		ppsidOwner,
		ppsidGroup,
		uintptr(unsafe.Pointer(ppDacl)),
		uintptr(unsafe.Pointer(ppSacl)),
		ppSecurityDescriptor,
		0,
	)
	// cf. https://msdn.microsoft.com/en-us/library/windows/desktop/aa446645(v=vs.85).aspx
	// If the function succeeds, the return value is ERROR_SUCCESS.
	// If the function fails, the return value is a nonzero error code defined in WinError.h.
	if r1 != 0 {
		err = syscall.Errno(r1)
	}
	return
}

func SetNamedSecurityInfo(
	objectName *uint16,
	objectType uint32,
	securityInfo uint32,
	psidOwner uintptr,
	psidGroup uintptr,
	pDacl *ACL,
	pSacl *ACL,
) (err error) {
	r1, _, _ := syscall.Syscall9(
		procSetNamedSecurityInfoW.Addr(),
		7,
		uintptr(unsafe.Pointer(objectName)),
		uintptr(objectType),
		uintptr(securityInfo),
		psidOwner,
		psidGroup,
		uintptr(unsafe.Pointer(pDacl)),
		uintptr(unsafe.Pointer(pSacl)),
		0, 0,
	)
	// cf. https://msdn.microsoft.com/en-us/library/windows/desktop/aa379579(v=vs.85).aspx
	// If the function succeeds, the return value is ERROR_SUCCESS.
	// If the function fails, the return value is a nonzero error code defined in WinError.h.
	if r1 != 0 {
		err = syscall.Errno(r1)
	}
	return
}

// TRUSTEE_FORM, cf. https://msdn.microsoft.com/en-us/library/windows/desktop/aa379638(v=vs.85).aspx
// do not reorder
const (
	TRUSTEE_IS_SID = iota
	TRUSTEE_IS_NAME
	TRUSTEE_BAD_FORM
	TRUSTEE_IS_OBJECTS_AND_SID
	TRUSTEE_IS_OBJECTS_AND_NAME
)

// struct _EXPLICIT_ACCESS, cf. https://msdn.microsoft.com/en-us/library/windows/desktop/aa446627(v=vs.85).aspx
type ExplicitAccess struct {
	AccessPermissions uint32
	AccessMode        uint32 // ACCESS_MODE
	Inheritance       uint32
	Trustee           Trustee
}

// dwInheritance flags in EXPLICIT_ACCESS
const (
	NO_INHERITANCE           = 0
	OBJECT_INHERIT_ACE       = 1 // (OI)
	CONTAINER_INHERIT_ACE    = 2 // (CI)
	NO_PROPAGATE_INHERIT_ACE = 4
)

// MULTIPLE_TRUSTEE_OPERATION enum, cf. https://msdn.microsoft.com/en-us/library/windows/desktop/aa379284(v=vs.85).aspx
// do not reorder.
const (
	NO_MULTIPLE_TRUSTEE = iota
	TRUSTEE_IS_IMPERSONATE
)

// TRUSTEE_TYPE enum, cf. https://msdn.microsoft.com/en-us/library/windows/desktop/aa379639(v=vs.85).aspx
const (
	TRUSTEE_IS_UNKNOWN = iota
	TRUSTEE_IS_USER
	TRUSTEE_IS_GROUP
	TRUSTEE_IS_DOMAIN
	TRUSTEE_IS_ALIAS
	TRUSTEE_IS_WELL_KNOWN_GROUP
	TRUSTEE_IS_DELETED
	TRUSTEE_IS_INVALID
	TRUSTEE_IS_COMPUTER
)

// struct _TRUSTEE, cf. https://msdn.microsoft.com/en-us/library/windows/desktop/aa379636(v=vs.85).aspx
type Trustee struct {
	MultipleTrustee          *Trustee
	MultipleTrusteeOperation uint32 // MULTIPLE_TRUSTEE_OPERATION
	TrusteeForm              uint32 // TRUSTEE_FORM
	TrusteeType              uint32 // TRUSTEE_TYPE
	Name                     *uint16
}

func SetEntriesInAcl(
	countOfExplicitEntries uint32,
	listOfExplicitEntries uintptr,
	oldAcl *ACL,
	newAcl **ACL,
) (err error) {
	r1, _, _ := syscall.Syscall6(
		procSetEntriesInAclW.Addr(),
		4,
		uintptr(countOfExplicitEntries),
		listOfExplicitEntries,
		uintptr(unsafe.Pointer(oldAcl)),
		uintptr(unsafe.Pointer(newAcl)),
		0, 0,
	)
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa379576(v=vs.85).aspx
	// If the function succeeds, the return value is ERROR_SUCCESS.
	// If the function fails, the return value is a nonzero error code defined in WinError.h.
	if r1 != 0 {
		err = syscall.Errno(r1)
	}
	return
}

// here be dragons
func MakeAbsoluteSD(
	pSelfRelativeSd uintptr,
	pAbsoluteSD uintptr,
	lpdwAbsoluteSDSize *uint32,
	pDacl *ACL,
	lpdwDaclSize *uint32,
	pSacl *ACL,
	lpdwSaclSize *uint32,
	pOwner uintptr,
	lpdwOwnerSize *uint32,
	pPrimaryGroup uintptr,
	lpdwPrimaryGroupSize *uint32,
) (err error) {
	r1, _, e1 := syscall.Syscall12(
		procMakeAbsoluteSD.Addr(),
		11,
		pSelfRelativeSd,
		pAbsoluteSD,
		uintptr(unsafe.Pointer(lpdwAbsoluteSDSize)),
		uintptr(unsafe.Pointer(pDacl)),
		uintptr(unsafe.Pointer(lpdwDaclSize)),
		uintptr(unsafe.Pointer(pSacl)),
		uintptr(unsafe.Pointer(lpdwSaclSize)),
		pOwner,
		uintptr(unsafe.Pointer(lpdwOwnerSize)),
		pPrimaryGroup,
		uintptr(unsafe.Pointer(lpdwPrimaryGroupSize)),
		0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func SetSecurityDescriptorDacl(
	pSecurityDescriptor uintptr,
	bDaclPresent uint32, // BOOL
	pDacl *ACL,
	bDaclDefaulted uint32, // BOOL
) (err error) {
	r1, _, e1 := syscall.Syscall6(
		procSetSecurityDescriptorDacl.Addr(),
		4,
		pSecurityDescriptor,
		uintptr(bDaclPresent),
		uintptr(unsafe.Pointer(pDacl)),
		uintptr(bDaclDefaulted),
		0, 0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func GetFileSecurity(
	fileName *uint16,
	requestedInformation uint32,
	pSecurityDescriptor uintptr,
	nLength uint32,
	nLengthNeeded *uint32,
) (err error) {
	r1, _, e1 := syscall.Syscall6(
		procGetFileSecurityW.Addr(),
		5,
		uintptr(unsafe.Pointer(fileName)),
		uintptr(requestedInformation),
		pSecurityDescriptor,
		uintptr(nLength),
		uintptr(unsafe.Pointer(nLengthNeeded)),
		0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func SetFileSecurity(
	fileName *uint16,
	securityInformation uint32,
	pSecurityDescriptor uintptr,
) (err error) {
	r1, _, e1 := syscall.Syscall(
		procSetFileSecurityW.Addr(),
		3,
		uintptr(unsafe.Pointer(fileName)),
		uintptr(securityInformation),
		pSecurityDescriptor,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

// struct _GENERIC_MAPPING
// cf. https://msdn.microsoft.com/en-us/library/windows/desktop/aa446633(v=vs.85).aspx
type GenericMapping struct {
	GenericRead    uint32
	GenericWrite   uint32
	GenericExecute uint32
	GenericAll     uint32
}

func AccessCheck(
	securityDescriptor uintptr,
	clientToken syscall.Token,
	desiredAccess uint32,
	genericMapping *GenericMapping,
	privilegeSet uintptr,
	privilegeSetLength *uint32,
	grantedAccess *uint32,
	accessStatus *bool,
) (err error) {
	r1, _, e1 := syscall.Syscall9(
		procAccessCheck.Addr(),
		8,
		securityDescriptor,
		uintptr(clientToken),
		uintptr(desiredAccess),
		uintptr(unsafe.Pointer(genericMapping)),
		privilegeSet,
		uintptr(unsafe.Pointer(privilegeSetLength)),
		uintptr(unsafe.Pointer(grantedAccess)),
		uintptr(unsafe.Pointer(accessStatus)),
		0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func MapGenericMask(
	accessMask *uint32,
	genericMapping *GenericMapping,
) {
	syscall.Syscall(
		procMapGenericMask.Addr(),
		2,
		uintptr(unsafe.Pointer(accessMask)),
		uintptr(unsafe.Pointer(genericMapping)),
		0,
	)
}
