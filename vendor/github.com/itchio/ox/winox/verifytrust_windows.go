package winox

import (
	"syscall"
	"unsafe"

	"github.com/itchio/ox/syscallex"
)

func VerifyTrust(path string) error {
	policyGUID := syscallex.WINTRUST_ACTION_GENERIC_VERIFY_V2

	fileData := new(syscallex.WinTrustFileInfo)
	fileData.CbStruct = uint32(unsafe.Sizeof(*fileData))
	fileData.FilePath = syscall.StringToUTF16Ptr(path)

	winTrustData := new(syscallex.WinTrustData)
	winTrustData.CbStruct = uint32(unsafe.Sizeof(*winTrustData))
	winTrustData.UIChoice = syscallex.WTD_UI_NONE
	winTrustData.RevocationChecks = syscallex.WTD_REVOKE_NONE
	winTrustData.UnionChoice = syscallex.WTD_CHOICE_FILE
	winTrustData.StateAction = syscallex.WTD_STATEACTION_VERIFY
	winTrustData.FileOrCatalogOrBlobOrSgnrOrCert = uintptr(unsafe.Pointer(fileData))

	trustErr := syscallex.WinVerifyTrust(syscall.Handle(0), &policyGUID, winTrustData)

	winTrustData.StateAction = syscallex.WTD_STATEACTION_CLOSE
	syscallex.WinVerifyTrust(syscall.Handle(0), &policyGUID, winTrustData)

	return trustErr
}
