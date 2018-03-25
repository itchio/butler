package runner

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/itchio/butler/butlerd/messages"

	"github.com/itchio/butler/installer"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/elevate"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/winsandbox"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/runner/execas"
	"github.com/itchio/butler/runner/syscallex"
	"github.com/itchio/butler/runner/winutil"
	"github.com/itchio/wharf/state"
)

type winsandboxRunner struct {
	params *RunnerParams

	playerData *winsandbox.PlayerData
}

var _ Runner = (*winsandboxRunner)(nil)

func newWinSandboxRunner(params *RunnerParams) (Runner, error) {
	wr := &winsandboxRunner{
		params: params,
	}
	return wr, nil
}

func (wr *winsandboxRunner) Prepare() error {
	consumer := wr.params.RequestContext.Consumer

	nullConsumer := &state.Consumer{}
	err := winsandbox.Check(nullConsumer)
	if err != nil {
		consumer.Warnf("Sandbox check failed: %s", err.Error())

		r, err := messages.AllowSandboxSetup.Call(wr.params.RequestContext, &butlerd.AllowSandboxSetupParams{})
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if !r.Allow {
			return &butlerd.ErrAborted{}
		}
		consumer.Infof("Proceeding with sandbox setup...")

		res, err := installer.RunSelf(&installer.RunSelfParams{
			Consumer: consumer,
			Args: []string{
				"--elevate",
				"winsandbox",
				"setup",
			},
		})
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if res.ExitCode != 0 {
			if res.ExitCode == elevate.ExitCodeAccessDenied {
				return &butlerd.ErrAborted{}
			}
		}

		err = installer.CheckExitCode(res.ExitCode, err)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		consumer.Infof("Sandbox setup done, checking again...")
		err = winsandbox.Check(nullConsumer)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	playerData, err := winsandbox.GetPlayerData()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	wr.playerData = playerData

	consumer.Infof("Sandbox is ready")
	return nil
}

func (wr *winsandboxRunner) Run() error {
	var err error
	params := wr.params
	consumer := params.RequestContext.Consumer
	pd := wr.playerData

	consumer.Infof("Running as user (%s)", pd.Username)

	env, err := wr.getEnvironment()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	sp, err := wr.getSharingPolicy()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Infof("Sharing policy: %s", sp)

	err = sp.Grant(consumer)
	if err != nil {
		comm.Warnf(err.Error())
		comm.Warnf("Attempting launch anyway...")
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
		return errors.Wrap(err, 0)
	}

	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = pg.AfterStart()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	// ok that SysProcAttr thing is 110% a hack but who are you
	// to judge me and how did you get into my home
	_, err = syscallex.ResumeThread(cmd.SysProcAttr.ThreadHandle)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = pg.Wait()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func (wr *winsandboxRunner) getSharingPolicy() (*winutil.SharingPolicy, error) {
	params := wr.params
	pd := wr.playerData
	consumer := params.RequestContext.Consumer

	sp := &winutil.SharingPolicy{
		Trustee: pd.Username,
	}

	impersonationToken, err := winutil.GetImpersonationToken(pd.Username, ".", pd.Password)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	defer winutil.SafeRelease(uintptr(impersonationToken))

	hasAccess, err := winutil.UserHasPermission(
		impersonationToken,
		syscallex.GENERIC_ALL,
		params.InstallFolder,
	)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	if !hasAccess {
		sp.Entries = append(sp.Entries, &winutil.ShareEntry{
			Path:        params.InstallFolder,
			Inheritance: winutil.InheritanceModeFull,
			Rights:      winutil.RightsFull,
		})
	}

	// cf. https://github.com/itchio/itch/issues/1470
	current := filepath.Dir(params.InstallFolder)
	for i := 0; i < 128; i++ { // dumb failsafe
		consumer.Debugf("Checking access for (%s)...", current)
		hasAccess, err := winutil.UserHasPermission(
			impersonationToken,
			syscallex.GENERIC_READ,
			current,
		)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		if !hasAccess {
			consumer.Debugf("Will need to grant temporary read permission to (%s)", current)
			sp.Entries = append(sp.Entries, &winutil.ShareEntry{
				Path:        current,
				Inheritance: winutil.InheritanceModeNone,
				Rights:      winutil.RightsRead,
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

func (wr *winsandboxRunner) getEnvironment() ([]string, error) {
	params := wr.params
	pd := wr.playerData

	env := params.Env
	setEnv := func(key string, value string) {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	setEnv("username", pd.Username)
	// we're not setting `userdomain` or `userdomain_roaming_profile`,
	// since we expect those to be the same for the regular user
	// and the sandbox user

	err := winutil.Impersonate(pd.Username, ".", pd.Password, func() error {
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
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return env, nil
}
