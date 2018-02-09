package syscallex

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modnetapi32 = windows.NewLazySystemDLL("netapi32.dll")

	procNetUserAdd = modnetapi32.NewProc("NetUserAdd")
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

const NERR_Success = 0

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
	switch r1 {
	case NERR_Success:
		// all good!
	case uintptr(syscall.ERROR_ACCESS_DENIED):
		err = syscall.ERROR_ACCESS_DENIED
	default:
		// must be a net error
		err = NERR(r1)
	}
	return
}

// NET_API_STATUS values
// cf. http://www.cs.uofs.edu/~beidler/Ada/win32/win32-lmerr.html
