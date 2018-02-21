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

char *GetLibraryPath() {
	NSArray* paths = NSSearchPathForDirectoriesInDomains(NSLibraryDirectory, NSUserDomainMask, YES);
    for (NSString* path in paths) {
		const char *tempString = [path UTF8String];
		char *ret = malloc(strlen(tempString) + 1);
		memcpy(ret, tempString, strlen(tempString) + 1);
		return ret;
	}
	return 0;
}

char *GetApplicationSupportPath() {
	NSArray* paths = NSSearchPathForDirectoriesInDomains(NSApplicationSupportDirectory, NSUserDomainMask, YES);
    for (NSString* path in paths) {
		const char *tempString = [path UTF8String];
		char *ret = malloc(strlen(tempString) + 1);
		memcpy(ret, tempString, strlen(tempString) + 1);
		return ret;
	}
	return 0;
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

func GetLibraryPath() (string, error) {
	cPath := C.GetLibraryPath()
	if uintptr(unsafe.Pointer(cPath)) == 0 {
		return "", fmt.Errorf("Could not get library path")
	}
	defer C.free(unsafe.Pointer(cPath))

	return C.GoString(cPath), nil
}

func GetApplicationSupportPath() (string, error) {
	cPath := C.GetApplicationSupportPath()
	if uintptr(unsafe.Pointer(cPath)) == 0 {
		return "", fmt.Errorf("Could not get application support path")
	}
	defer C.free(unsafe.Pointer(cPath))

	return C.GoString(cPath), nil
}
