// +build windows

package runner

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/runner/execas"
	"github.com/itchio/butler/runner/syscallex"
	"github.com/itchio/butler/runner/winutil"

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
	var err error
	params := wr.params
	consumer := params.Consumer

	consumer.Infof("Running as user (%s)", wr.username)

	env := params.Env
	setEnv := func(key string, value string) {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	setEnv("username", wr.username)
	// we're not setting `userdomain` or `userdomain_roaming_profile`,
	// since we expect those to be the same for the regular user
	// and the sandbox user

	err = winutil.Impersonate(wr.username, ".", wr.password, func() error {
		profileDir, err := winutil.GetFolderPath(winutil.FolderTypeProfile)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		// environment variables are case-insensitive on windows,
		// and exec{,as}.Command do case-insensitive deduplication properly
		setEnv("userprofile", profileDir)

		// when %userprofile% is `C:\Users\terry`,
		// %homepath% is usually `\Users\terry`.
		homePath := strings.TrimPrefix(profileDir, filepath.VolumeName(profileDir))
		setEnv("homepath", homePath)

		appDataDir, err := winutil.GetFolderPath(winutil.FolderTypeAppData)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		setEnv("appdata", appDataDir)

		localAppDataDir, err := winutil.GetFolderPath(winutil.FolderTypeLocalAppData)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		setEnv("localappdata", localAppDataDir)

		return nil
	})

	err = SetupJobObject(consumer)
	if err != nil {
		return errors.Wrap(err, 0)
	}

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
