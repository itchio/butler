// +build windows

package winox

import (
	"syscall"
	"unsafe"

	"github.com/itchio/ox/syscallex"
	"github.com/pkg/errors"
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
	default:
		return "", errors.Errorf("Unknown folder type: %d", folderType)
	}
	csidl |= syscallex.CSIDL_FLAG_CREATE

	ret, err := syscallex.SHGetFolderPath(
		0,
		csidl,
		0,
		syscallex.SHGFP_TYPE_CURRENT,
	)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return ret, nil
}

type ImpersonateCallback func() error

func Logon(username string, domain string, password string) (syscall.Token, error) {
	var token syscall.Token
	err := syscallex.LogonUser(
		syscall.StringToUTF16Ptr(username),
		syscall.StringToUTF16Ptr(domain),
		syscall.StringToUTF16Ptr(password),
		syscallex.LOGON32_LOGON_INTERACTIVE,
		syscallex.LOGON32_PROVIDER_DEFAULT,
		&token,
	)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return token, nil
}

func Impersonate(username string, domain string, password string, cb ImpersonateCallback) error {
	token, err := Logon(username, domain, password)
	if err != nil {
		return errors.WithStack(err)
	}
	defer syscall.CloseHandle(syscall.Handle(token))

	_, err = syscall.GetEnvironmentStrings()
	if err != nil {
		return errors.WithStack(err)
	}

	err = syscallex.ImpersonateLoggedOnUser(token)
	if err != nil {
		return errors.WithStack(err)
	}

	defer syscallex.RevertToSelf()

	return cb()
}

func AddUser(username string, password string, comment string) error {
	var usri1 = syscallex.UserInfo1{
		Name:     syscall.StringToUTF16Ptr(username),
		Password: syscall.StringToUTF16Ptr(password),
		Priv:     syscallex.USER_PRIV_USER,
		Flags:    syscallex.UF_SCRIPT,
		Comment:  syscall.StringToUTF16Ptr(comment),
	}

	err := syscallex.NetUserAdd(
		nil,
		1,
		uintptr(unsafe.Pointer(&usri1)),
		nil,
	)
	if err != nil {
		return errors.WithMessage(err, "NetUserAdd")
	}

	return nil
}

func ForceSetPassword(username string, password string) error {
	var usri1003 = syscallex.UserInfo1003{
		Password: syscall.StringToUTF16Ptr(password),
	}

	err := syscallex.NetUserSetInfo(
		nil,
		syscall.StringToUTF16Ptr(username),
		1003,
		uintptr(unsafe.Pointer(&usri1003)),
		nil,
	)
	if err != nil {
		return errors.WithMessage(err, "NetUserSetInfo (password)")
	}

	return nil
}

// Remove "username" from the "Users" group if needed
func RemoveUserFromUsersGroup(username string) error {
	var arbitrarySize = 2048
	var sidSize uint32 = uint32(arbitrarySize)
	sid := make([]byte, sidSize)

	err := syscallex.CreateWellKnownSid(
		syscallex.WinBuiltinUsersSid,
		0, // domainSid
		uintptr(unsafe.Pointer(&sid[0])),
		&sidSize,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	var cchName = uint32(arbitrarySize)
	name := make([]uint16, cchName)

	var cchReferencedDomainName = uint32(arbitrarySize)
	referencedDomainName := make([]uint16, cchReferencedDomainName)

	var sidUse uint32

	err = syscallex.LookupAccountSid(
		nil, // systemName
		uintptr(unsafe.Pointer(&sid[0])),
		&name[0],
		&cchName,
		&referencedDomainName[0],
		&cchReferencedDomainName,
		&sidUse,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	usersGroupName := &name[0]

	var gmi [1]syscallex.LocalGroupMembersInfo3
	gmi[0].DomainAndName = syscall.StringToUTF16Ptr(username)

	err = syscallex.NetLocalGroupDelMembers(
		nil,            // servername
		usersGroupName, // groupName
		3,              // level
		uintptr(unsafe.Pointer(&gmi[0])),
		1, // totalentries
	)
	if err != nil {
		if en, ok := err.(syscall.Errno); ok {
			if en == syscallex.ERROR_MEMBER_NOT_IN_ALIAS {
				// User wasn't in Users group. That's ok!
				return nil
			}
		}
		return errors.WithStack(err)
	}

	return nil
}

func LoadProfileOnce(username string, domain string, password string) error {
	token, err := Logon(username, password, password)
	if err != nil {
		return errors.WithStack(err)
	}

	defer syscall.CloseHandle(syscall.Handle(token))

	var profileInfo syscallex.ProfileInfo
	profileInfo.Size = uint32(unsafe.Sizeof(profileInfo))
	profileInfo.UserName = syscall.StringToUTF16Ptr(username)
	profileInfo.Flags = syscallex.PI_NOUI

	err = syscallex.LoadUserProfile(token, &profileInfo)
	if err != nil {
		return errors.WithStack(err)
	}

	err = syscallex.UnloadUserProfile(token, profileInfo.Profile)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

type causer interface {
	Cause() error
}

func AsErrno(err error) (syscall.Errno, bool) {
	if se, ok := err.(causer); ok {
		return AsErrno(se.Cause())
	}

	en, ok := err.(syscall.Errno)
	return en, ok
}
