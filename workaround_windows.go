package main

/*
#cgo LDFLAGS: -luser32
#include <windows.h>
*/
import "C"

func init() {
	// this is the most harmless user32.dll call I could find
	// that forces gcc on our windows CI worker to link against
	// it. If you remove this, windows sandboxing will break.
	// No, I don't know either. It seems a Windows 2012 update
	// broke this.
	C.IsWindow(C.HWND(nil));
}

