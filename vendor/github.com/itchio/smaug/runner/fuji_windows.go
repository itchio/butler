//+build windows

package runner

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/itchio/ox/syscallex"
	"github.com/itchio/ox/winox"
	"github.com/itchio/ox/winox/execas"
	"github.com/itchio/smaug/fuji"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type fujiRunner struct {
	params      *RunnerParams
	Credentials *fuji.Credentials
}

var _ Runner = (*fujiRunner)(nil)

func newFujiRunner(params *RunnerParams) (Runner, error) {
	if params.FujiParams.Instance == nil {
		return nil, errors.Errorf("FujiParams.Instance should be set")
	}

	wr := &fujiRunner{
		params: params,
	}
	return wr, nil
}

func (wr *fujiRunner) Prepare() error {
	consumer := wr.params.Consumer
	fi := wr.params.FujiParams.Instance

	nullConsumer := &state.Consumer{}
	err := fi.Check(nullConsumer)
	if err != nil {
		consumer.Warnf("Sandbox check failed: %s", err.Error())

		err := wr.params.FujiParams.PerformElevatedSetup()
		if err != nil {
			return err
		}

		consumer.Infof("Sandbox setup done, checking again...")
		err = fi.Check(nullConsumer)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	Credentials, err := fi.GetCredentials()
	if err != nil {
		return errors.WithStack(err)
	}

	wr.Credentials = Credentials

	consumer.Infof("Sandbox is ready")
	return nil
}

func (wr *fujiRunner) Run() error {
	var err error
	params := wr.params
	consumer := params.Consumer
	pd := wr.Credentials

	consumer.Infof("Running as user (%s)", pd.Username)

	env, err := wr.getEnvironment()
	if err != nil {
		return errors.WithStack(err)
	}

	sp, err := wr.getSharingPolicy()
	if err != nil {
		return errors.WithStack(err)
	}

	consumer.Infof("Sharing policy: %s", sp)

	err = sp.Grant(consumer)
	if err != nil {
		consumer.Warnf(err.Error())
		consumer.Warnf("Attempting launch anyway...")
	}

	defer sp.Revoke(consumer)

	cmd := execas.Command(params.FullTargetPath, params.Args...)
	cmd.Username = pd.Username
	cmd.Domain = "."
	cmd.Password = pd.Password
	cmd.Dir = params.Dir
	cmd.Env = env
	cmd.Stdout = params.Stdout
	cmd.Stderr = params.Stderr

	cmd.SysProcAttr = &syscallex.SysProcAttr{
		CreationFlags: syscallex.CREATE_SUSPENDED,
		LogonFlags:    syscallex.LOGON_WITH_PROFILE,
	}

	pg, err := NewProcessGroup(consumer, cmd, params.Ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	err = cmd.Start()
	if err != nil {
		return errors.WithStack(err)
	}

	err = pg.AfterStart()
	if err != nil {
		return errors.WithStack(err)
	}

	err = pg.Wait()
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (wr *fujiRunner) getSharingPolicy() (*winox.SharingPolicy, error) {
	params := wr.params
	pd := wr.Credentials
	consumer := params.Consumer

	sp := &winox.SharingPolicy{
		Trustee: pd.Username,
	}

	impersonationToken, err := winox.GetImpersonationToken(pd.Username, ".", pd.Password)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer winox.SafeRelease(uintptr(impersonationToken))

	hasAccess, err := winox.UserHasPermission(
		impersonationToken,
		syscallex.GENERIC_ALL,
		params.InstallFolder,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if !hasAccess {
		sp.Entries = append(sp.Entries, &winox.ShareEntry{
			Path:        params.InstallFolder,
			Inheritance: winox.InheritanceModeFull,
			Rights:      winox.RightsFull,
		})
	}

	// cf. https://github.com/itchio/itch/issues/1470
	current := filepath.Dir(params.InstallFolder)
	for i := 0; i < 128; i++ { // dumb failsafe
		consumer.Debugf("Checking access for (%s)...", current)
		hasAccess, err := winox.UserHasPermission(
			impersonationToken,
			syscallex.GENERIC_READ,
			current,
		)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if !hasAccess {
			consumer.Debugf("Will need to grant temporary read permission to (%s)", current)
			sp.Entries = append(sp.Entries, &winox.ShareEntry{
				Path:        current,
				Inheritance: winox.InheritanceModeNone,
				Rights:      winox.RightsRead,
			})
		}
		next := filepath.Dir(current)
		if next == current {
			break
		}
		current = next
	}

	return sp, nil
}

func (wr *fujiRunner) getEnvironment() ([]string, error) {
	params := wr.params
	pd := wr.Credentials

	env := params.Env
	setEnv := func(key string, value string) {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	setEnv("username", pd.Username)
	// we're not setting `userdomain` or `userdomain_roaming_profile`,
	// since we expect those to be the same for the regular user
	// and the sandbox user

	err := winox.Impersonate(pd.Username, ".", pd.Password, func() error {
		profileDir, err := winox.GetFolderPath(winox.FolderTypeProfile)
		if err != nil {
			return errors.WithStack(err)
		}
		// environment variables are case-insensitive on windows,
		// and exec{,as}.Command do case-insensitive deduplication properly
		setEnv("userprofile", profileDir)

		// when %userprofile% is `C:\Users\terry`,
		// %homepath% is usually `\Users\terry`.
		homePath := strings.TrimPrefix(profileDir, filepath.VolumeName(profileDir))
		setEnv("homepath", homePath)

		appDataDir, err := winox.GetFolderPath(winox.FolderTypeAppData)
		if err != nil {
			return errors.WithStack(err)
		}
		setEnv("appdata", appDataDir)

		localAppDataDir, err := winox.GetFolderPath(winox.FolderTypeLocalAppData)
		if err != nil {
			return errors.WithStack(err)
		}
		setEnv("localappdata", localAppDataDir)

		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return env, nil
}
