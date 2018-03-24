package syscallex

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	SHGFP_TYPE_CURRENT = 0
)

// see http://svnpenn.blogspot.com/2011/01/csidl-constants.html
const (
	CSIDL_FLAG_CREATE   = 0x8000
	CSIDL_APPDATA       = 0x001a
	CSIDL_PROFILE       = 0x0028
	CSIDL_LOCAL_APPDATA = 0x001c
)

const MAX_PATH = 260

var (
	modshell32 = windows.NewLazySystemDLL("shell32.dll")

	procSHGetFolderPathW = modshell32.NewProc("SHGetFolderPathW")
)

func SHGetFolderPath(
	owner syscall.Handle,
	folder uint32,
	token syscall.Token,
	flags uint32,
) (s string, err error) {
	buffer := make([]uint16, MAX_PATH+1)

	r1, _, e1 := syscall.Syscall6(
		procSHGetFolderPathW.Addr(),
		5,
		uintptr(owner),
		uintptr(folder),
		uintptr(token),
		uintptr(flags),
		uintptr(unsafe.Pointer(&buffer[0])),
		0,
	)
	if FAILED(r1) {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	if err == nil {
		s = syscall.UTF16ToString(buffer)
	}
	return
}

const ERROR_SUCCESS = 0

func FAILED(r1 uintptr) bool {
	return r1 != ERROR_SUCCESS
}
