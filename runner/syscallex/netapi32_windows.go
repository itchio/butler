package syscallex

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modnetapi32 = windows.NewLazySystemDLL("netapi32.dll")

	procNetUserAdd              = modnetapi32.NewProc("NetUserAdd")
	procNetUserSetInfo          = modnetapi32.NewProc("NetUserSetInfo")
	procNetLocalGroupDelMembers = modnetapi32.NewProc("NetLocalGroupDelMembers")
	procNetUserChangePassword   = modnetapi32.NewProc("NetUserChangePassword")
)

// struct _USER_INFO_1, cf.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa371109(v=vs.85).aspx
type UserInfo1 struct {
	Name        *uint16
	Password    *uint16
	PasswordAge uint32
	Priv        uint32
	HomeDir     *uint16
	Comment     *uint16
	Flags       uint32
	ScriptPath  *uint16
}

// struct _USER_INFO_1003, cf.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa370963(v=vs.85).aspx
type UserInfo1003 struct {
	Password *uint16
}

// struct LOCALGROUP_MEMBERS_INFO_3, cf.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa370281(v=vs.85).aspx
type LocalGroupMembersInfo3 struct {
	DomainAndName *uint16
}

// see http://www.rensselaer.org/dept/cis/software/g77-mingw32/include/lmaccess.h
const (
	UF_SCRIPT = 1
)

const (
	USER_PRIV_GUEST = 0
	USER_PRIV_USER  = 1
	USER_PRIV_ADMIN = 2
)

func NetUserAdd(
	servername *uint16,
	level uint32,
	buf uintptr,
	parmErr *uint32,
) (err error) {
	r1, _, _ := syscall.Syscall6(
		procNetUserAdd.Addr(),
		4,
		uintptr(unsafe.Pointer(servername)),
		uintptr(level),
		buf,
		uintptr(unsafe.Pointer(parmErr)),
		0, 0,
	)
	if r1 != 0 {
		err = syscall.Errno(r1)
	}
	return
}

func NetUserSetInfo(
	servername *uint16,
	username *uint16,
	level uint32,
	buf uintptr,
	parmErr *uint32,
) (err error) {
	r1, _, _ := syscall.Syscall6(
		procNetUserSetInfo.Addr(),
		5,
		uintptr(unsafe.Pointer(servername)),
		uintptr(unsafe.Pointer(username)),
		uintptr(level),
		buf,
		uintptr(unsafe.Pointer(parmErr)),
		0,
	)
	if r1 != 0 {
		err = syscall.Errno(r1)
	}
	return
}

func NetLocalGroupDelMembers(
	servername *uint16,
	groupname *uint16,
	level uint32,
	buf uintptr,
	totalentries uint32,
) (err error) {
	r1, _, _ := syscall.Syscall6(
		procNetLocalGroupDelMembers.Addr(),
		5,
		uintptr(unsafe.Pointer(servername)),
		uintptr(unsafe.Pointer(groupname)),
		uintptr(level),
		buf,
		uintptr(totalentries),
		0,
	)
	if r1 != 0 {
		err = syscall.Errno(r1)
	}
	return
}

// cf. https://www.rpi.edu/dept/cis/software/g77-mingw32/include/winerror.h
const (
	ERROR_INVALID_PASSWORD     syscall.Errno = 86
	ERROR_PASSWORD_EXPIRED     syscall.Errno = 1330
	ERROR_PASSWORD_MUST_CHANGE syscall.Errno = 1907
	ERROR_MEMBER_NOT_IN_ALIAS  syscall.Errno = 1377
)

func NetUserChangePassword(
	domainname *uint16,
	username *uint16,
	oldpassword *uint16,
	newpassword *uint16,
) (err error) {
	r1, _, _ := syscall.Syscall6(
		procNetUserChangePassword.Addr(),
		4,
		uintptr(unsafe.Pointer(domainname)),
		uintptr(unsafe.Pointer(username)),
		uintptr(unsafe.Pointer(oldpassword)),
		uintptr(unsafe.Pointer(newpassword)),
		0, 0,
	)
	if r1 != 0 {
		err = syscall.Errno(r1)
	}
	return
}
