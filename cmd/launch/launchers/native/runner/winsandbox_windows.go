// +build windows

package runner

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/launch/launchers/native/runner/syscallex"

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

	domain := "."

	si := new(syscall.StartupInfo)
	si.Cb = uint32(unsafe.Sizeof(*si))

	pi := new(syscall.ProcessInformation)

	err := syscallex.CreateProcessWithLogon(
		syscall.StringToUTF16Ptr(wr.username),
		syscall.StringToUTF16Ptr(domain),
		syscall.StringToUTF16Ptr(wr.password),
		syscallex.LOGON_WITH_PROFILE,
		syscall.StringToUTF16Ptr(wr.params.FullTargetPath),
		nil, // commandLine
		0,   // creationFlags
		nil, // env
		syscall.StringToUTF16Ptr(wr.params.Dir),
		si,
		pi,
	)
	if err != nil {
		return errors.Wrap(err, 0)
	}

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
