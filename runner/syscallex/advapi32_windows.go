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

var (
	modadvapi32 = windows.NewLazySystemDLL("advapi32.dll")

	procCreateProcessWithLogonW = modadvapi32.NewProc("CreateProcessWithLogonW")
	procLogonUserW              = modadvapi32.NewProc("LogonUserW")
	procImpersonateLoggedOnUser = modadvapi32.NewProc("ImpersonateLoggedOnUser")
	procRevertToSelf            = modadvapi32.NewProc("RevertToSelf")
	procLookupAccountNameW      = modadvapi32.NewProc("LookupAccountNameW")
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
	outToken *syscall.Handle,
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
	token syscall.Handle,
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
