package syscallex

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	moduser32 = windows.NewLazySystemDLL("user32.dll")

	procEnumWindows              = moduser32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = moduser32.NewProc("GetWindowThreadProcessId")
	procSetForegroundWindow      = moduser32.NewProc("SetForegroundWindow")
	procShowWindow               = moduser32.NewProc("ShowWindow")
	procIsWindowVisible          = moduser32.NewProc("IsWindowVisible")
	procSwitchToThisWindow       = moduser32.NewProc("SwitchToThisWindow")
)

func EnumWindows(
	cb uintptr,
	lparam uintptr,
) (err error) {
	r1, _, e1 := syscall.Syscall(
		procEnumWindows.Addr(),
		2,
		cb,
		lparam,
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

func GetWindowThreadProcessId(
	hwnd syscall.Handle,
	pProcessId *uint32,
) uint32 {
	r1, _, _ := syscall.Syscall(
		procGetWindowThreadProcessId.Addr(),
		2,
		uintptr(hwnd),
		uintptr(unsafe.Pointer(pProcessId)),
		0,
	)
	return uint32(r1)
}

func SetForegroundWindow(
	hwnd syscall.Handle,
) (err error) {
	r1, _, e1 := syscall.Syscall(
		procSetForegroundWindow.Addr(),
		1,
		uintptr(hwnd),
		0,
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

func ShowWindow(
	hwnd syscall.Handle,
	flags int,
) (err error) {
	r1, _, e1 := syscall.Syscall(
		procShowWindow.Addr(),
		2,
		uintptr(hwnd),
		uintptr(flags),
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

func IsWindowVisible(
	hwnd syscall.Handle,
) bool {
	ret, _, _ := syscall.Syscall(
		procIsWindowVisible.Addr(),
		1,
		uintptr(hwnd),
		0,
		0,
	)

	return ret != 0
}

func SwitchToThisWindow(
	hwnd syscall.Handle,
	altTab bool,
) {
	altTabInt := 0
	if altTab {
		altTabInt = 1
	}

	syscall.Syscall(
		procSwitchToThisWindow.Addr(),
		2,
		uintptr(hwnd),
		uintptr(altTabInt),
		0,
	)
}
