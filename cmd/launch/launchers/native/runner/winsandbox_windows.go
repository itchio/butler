// +build windows

package runner

import (
	"fmt"
	"syscall"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/launch/launchers/native/runner/execas"
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
	params := wr.params
	consumer := params.Consumer

	err := SetupJobObject(consumer)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Infof("Running as user (%s)", wr.username)

	var token syscall.Handle
	err = syscallex.LogonUser(
		syscall.StringToUTF16Ptr(wr.username),
		syscall.StringToUTF16Ptr("."),
		syscall.StringToUTF16Ptr(wr.password),
		syscallex.LOGON32_LOGON_INTERACTIVE,
		syscallex.LOGON32_PROVIDER_DEFAULT,
		&token,
	)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	_, err = syscall.GetEnvironmentStrings()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = syscallex.ImpersonateLoggedOnUser(token)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	profileDir, err := syscallex.SHGetFolderPath(
		0,
		syscallex.CSIDL_PROFILE|syscallex.CSIDL_FLAG_CREATE,
		0,
		syscallex.SHGFP_TYPE_CURRENT,
	)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	appDataDir, err := syscallex.SHGetFolderPath(
		0,
		syscallex.CSIDL_APPDATA|syscallex.CSIDL_FLAG_CREATE,
		0,
		syscallex.SHGFP_TYPE_CURRENT,
	)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	localAppDataDir, err := syscallex.SHGetFolderPath(
		0,
		syscallex.CSIDL_LOCAL_APPDATA|syscallex.CSIDL_FLAG_CREATE,
		0,
		syscallex.SHGFP_TYPE_CURRENT,
	)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = syscallex.RevertToSelf()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	env := params.Env
	env = append(env, fmt.Sprintf("userprofile=%s", profileDir))
	env = append(env, fmt.Sprintf("appdata=%s", appDataDir))
	env = append(env, fmt.Sprintf("localappdata=%s", localAppDataDir))

	cmd := execas.CommandContext(params.Ctx, params.FullTargetPath, params.Args...)
	cmd.Username = wr.username
	cmd.Domain = "."
	cmd.Password = wr.password
	cmd.Dir = params.Dir
	cmd.Env = env
	cmd.Stdout = params.Stdout
	cmd.Stderr = params.Stderr
	cmd.SysProcAttr = &syscallex.SysProcAttr{
		LogonFlags: syscallex.LOGON_WITH_PROFILE,
	}

	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = WaitJobObject(consumer)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
