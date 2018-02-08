// +build windows

package runner

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/go-errors/errors"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type winsandboxRunner struct {
	params *RunnerParams

	username string
	password string
}

var _ Runner = (*winsandboxRunner)(nil)

func newWinSandboxRunner(params *RunnerParams) (Runner, error) {
	wr := &winsandboxRunner{
		params: params,
	}
	return wr, nil
}

func (wr *winsandboxRunner) Prepare() error {
	// TODO: create user if it doesn't exist
	consumer := wr.params.Consumer

	username, err := wr.getItchPlayerData("username")
	if err != nil {
		return errors.Wrap(err, 0)
	}
	wr.username = username

	password, err := wr.getItchPlayerData("password")
	if err != nil {
		return errors.Wrap(err, 0)
	}
	wr.password = password

	consumer.Infof("Successfully retrieved login details for sandbox user")
	return nil
}

const itchPlayerRegistryKey = `SOFTWARE\itch\Sandbox`

func (wr *winsandboxRunner) getItchPlayerData(name string) (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, itchPlayerRegistryKey, registry.QUERY_VALUE)
	if err != nil {
		return "", errors.Wrap(err, 0)
	}

	defer key.Close()

	ret, _, err := key.GetStringValue(name)
	if err != nil {
		return "", errors.Wrap(err, 0)
	}

	return ret, nil
}

func (wr *winsandboxRunner) Run() error {
	consumer := wr.params.Consumer

	// just a quick POC, will be extracted into its own package
	var advapi = windows.NewLazyDLL("advapi32.dll")
	var createProcess = advapi.NewProc("CreateProcessWithLogonW")
	// var LOGON32_LOGON_INTERACTIVE = 2
	// var LOGON32_PROVIDER_DEFAULT = 0
	var LOGON_WITH_PROFILE = 1

	domain := "."

	si := new(syscall.StartupInfo)
	si.Cb = uint32(unsafe.Sizeof(*si))

	pi := new(syscall.ProcessInformation)

	ret, _, _ := createProcess.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(wr.username))),              // username
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(domain))),                   // domain
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(wr.password))),              // password
		uintptr(LOGON_WITH_PROFILE),                                                 // logon flags
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(wr.params.FullTargetPath))), // app name
		uintptr(0),                  // command line
		uintptr(0),                  // creation flags
		uintptr(0),                  // environment
		uintptr(0),                  // current directory
		uintptr(unsafe.Pointer(si)), // startup info
		uintptr(unsafe.Pointer(pi)), // process info
	)
	consumer.Infof("CreateProcess returned: %d", ret)
	consumer.Infof("Process ID: %d", pi.ProcessId)
	syscall.CloseHandle(pi.Thread)

	p, err := os.FindProcess(int(pi.ProcessId))
	if err != nil {
		return errors.Wrap(err, 0)
	}

	_, err = p.Wait()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
