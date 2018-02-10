// +build darwin

package macutil

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#include <stdlib.h>

char *GetExecutablePath(char *cBundlePath) {
    NSString* bundlePath = [NSString stringWithUTF8String:cBundlePath];
	if (!bundlePath) {
		return 0;
	}

	NSBundle* bundle = [NSBundle bundleWithPath:bundlePath];
	if (!bundle) {
		return 0;
	}

    const char *tempString = [[bundle executablePath] UTF8String];
    char *ret = malloc(strlen(tempString) + 1);
    memcpy(ret, tempString, strlen(tempString) + 1);
    return ret;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func GetExecutablePath(bundlePath string) (string, error) {
	cPath := C.GetExecutablePath(C.CString(bundlePath))
	if uintptr(unsafe.Pointer(cPath)) == 0 {
		return "", fmt.Errorf("Could not get executable path for app bundle (%s)", bundlePath)
	}
	defer C.free(unsafe.Pointer(cPath))

	return C.GoString(cPath), nil
}
