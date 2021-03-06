package butlerd

import "fmt"

var _ Error = Code(0)

var codeMessages = map[Code]string{
	CodeOperationCancelled: "The operation was cancelled.",
	CodeOperationAborted:   "The operation was aborted.",

	CodeInstallFolderDisappeared: "Launch was unsuccessful because install folder disappeared",

	CodeNoCompatibleUploads: "No compatible uploads were found.",

	CodeUnsupportedHost: "This title is hosted on an incompatible third-party website",

	CodeNoLaunchCandidates: "Nothing that can be launched was found.",

	CodeJavaRuntimeNeeded: "Java Runtime Environment is required to launch this title.",

	CodeNetworkDisconnected: "There is no Internet connection",

	CodeAPIError: "API error",

	CodeDatabaseBusy: "The database is busy",

	CodeCantRemoveLocationBecauseOfActiveDownloads: "An install location could not be removed because it has active downloads",
}

func (code Code) RpcErrorMessage() string {
	if msg, ok := codeMessages[code]; ok {
		return msg
	}
	return fmt.Sprintf("butlerd error %d", code)
}

func (code Code) RpcErrorCode() int64 {
	return int64(code)
}

func (code Code) RpcErrorData() map[string]interface{} {
	return nil
}

func (code Code) Error() string {
	return code.RpcErrorMessage()
}

func (code Code) String() string {
	return fmt.Sprintf("butlerd error: %s", code.Error())
}
