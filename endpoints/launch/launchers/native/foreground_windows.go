// +build windows

package native

import (
	"log"
	"syscall"

	"github.com/itchio/ox/syscallex"
)

func setWindowForeground(hwnd int64) {
	var err error
	err = syscallex.ShowWindow(syscall.Handle(hwnd), syscall.SW_MINIMIZE)
	if err != nil {
		log.Printf("ShowWindow error: %v", err)
	}
	err = syscallex.ShowWindow(syscall.Handle(hwnd), syscall.SW_RESTORE)
	if err != nil {
		log.Printf("ShowWindow error: %v", err)
	}
	err = syscallex.SetForegroundWindow(syscall.Handle(hwnd))
	if err != nil {
		log.Printf("SetForegroundWindow error: %v", err)
	}
}
