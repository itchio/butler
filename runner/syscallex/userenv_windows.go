// +build windows

package syscallex

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	moduserenv = windows.NewLazySystemDLL("userenv.dll")

	procLoadUserProfileW  = moduserenv.NewProc("LoadUserProfileW")
	procUnloadUserProfile = moduserenv.NewProc("UnloadUserProfile")
)

// flags for the ProfileInfo struct
const (
	// Prevents the display of profile error messages.
	PI_NOUI = 1
)

// struct _PROFILEINFO, cf.
// https://msdn.microsoft.com/en-us/library/windows/desktop/bb773378(v=vs.85).aspx
type ProfileInfo struct {
	Size        uint32
	Flags       uint32
	UserName    *uint16
	ProfilePath *uint16
	Defaultpath *uint16
	ServerName  *uint16
	PolicyPath  *uint16
	Profile     syscall.Handle
}

func LoadUserProfile(
	token syscall.Token,
	profileInfo *ProfileInfo,
) (err error) {
	r1, _, e1 := syscall.Syscall(
		procLoadUserProfileW.Addr(),
		2,
		uintptr(token),
		uintptr(unsafe.Pointer(profileInfo)),
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

func UnloadUserProfile(
	token syscall.Token,
	profile syscall.Handle,
) (err error) {
	r1, _, e1 := syscall.Syscall(
		procUnloadUserProfile.Addr(),
		2,
		uintptr(token),
		uintptr(profile),
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
