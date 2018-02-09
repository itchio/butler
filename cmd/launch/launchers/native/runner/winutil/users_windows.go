// +build windows

package winutil

import (
	"syscall"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/launch/launchers/native/runner/syscallex"
)

type FolderType int

const (
	FolderTypeProfile FolderType = iota
	FolderTypeAppData
	FolderTypeLocalAppData
)

func GetFolderPath(folderType FolderType) (string, error) {
	var csidl uint32
	switch folderType {
	case FolderTypeProfile:
		csidl = syscallex.CSIDL_PROFILE
	case FolderTypeAppData:
		csidl = syscallex.CSIDL_APPDATA
	case FolderTypeLocalAppData:
		csidl = syscallex.CSIDL_LOCAL_APPDATA
	}
	csidl |= syscallex.CSIDL_FLAG_CREATE

	ret, err := syscallex.SHGetFolderPath(
		0,
		csidl,
		0,
		syscallex.SHGFP_TYPE_CURRENT,
	)
	if err != nil {
		return "", errors.Wrap(err, 0)
	}
	return ret, nil
}

type ImpersonateCallback func() error

func Impersonate(username string, domain string, password string, cb ImpersonateCallback) error {
	var token syscall.Handle
	err := syscallex.LogonUser(
		syscall.StringToUTF16Ptr(username),
		syscall.StringToUTF16Ptr(domain),
		syscall.StringToUTF16Ptr(password),
		syscallex.LOGON32_LOGON_INTERACTIVE,
		syscallex.LOGON32_PROVIDER_DEFAULT,
		&token,
	)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	defer syscall.CloseHandle(token)

	_, err = syscall.GetEnvironmentStrings()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = syscallex.ImpersonateLoggedOnUser(token)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	defer syscallex.RevertToSelf()

	return cb()
}
