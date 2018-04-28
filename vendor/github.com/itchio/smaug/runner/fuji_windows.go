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
	fi          fuji.Instance
	credentials *fuji.Credentials
}

var _ Runner = (*fujiRunner)(nil)

func newFujiRunner(params *RunnerParams) (Runner, error) {
	if params.FujiParams.Settings == nil {
		return nil, errors.Errorf("FujiParams.Instance should be set")
	}

	fi, err := fuji.NewInstance(params.FujiParams.Settings)
	if err != nil {
		return nil, err
	}

	wr := &fujiRunner{
		params: params,
		fi:     fi,
	}
	return wr, nil
}

func (wr *fujiRunner) Prepare() error {
	consumer := wr.params.Consumer
	fi := wr.fi

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

	credentials, err := fi.GetCredentials()
	if err != nil {
		return errors.WithStack(err)
	}

	wr.credentials = credentials

	consumer.Infof("Sandbox is ready")
	return nil
}

func (wr *fujiRunner) Run() error {
	var err error
	params := wr.params
	consumer := params.Consumer
	creds := wr.credentials

	consumer.Infof("Running as user (%s)", creds.Username)

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
	cmd.Username = creds.Username
	cmd.Domain = "."
	cmd.Password = creds.Password
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
	creds := wr.credentials
	consumer := params.Consumer

	sp := &winox.SharingPolicy{
		Trustee: creds.Username,
	}

	impersonationToken, err := winox.GetImpersonationToken(creds.Username, ".", creds.Password)
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
	creds := wr.credentials

	env := params.Env
	setEnv := func(key string, value string) {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	setEnv("username", creds.Username)
	// we're not setting `userdomain` or `userdomain_roaming_profile`,
	// since we expect those to be the same for the regular user
	// and the sandbox user

	err := winox.Impersonate(creds.Username, ".", creds.Password, func() error {
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
