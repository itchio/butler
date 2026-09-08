// Package steamsync holds the butler commands that move a developer's
// Steam builds and store listing over to itch.io. The Steam side lives
// in package steam; this package is the CLI surface over it.
package steamsync

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/butler/steam"
	"github.com/itchio/fresh-steamer/session"
	"github.com/pkg/errors"
	"golang.org/x/term"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// Credentials given as flags win over the environment and the file.
var credArgs steam.Creds

func registerCredFlags(cmd *kingpin.CmdClause) {
	cmd.Flag("steam-refresh-token", "Steam login token to use instead of the stored one. Also read from "+steam.EnvRefreshToken+".").StringVar(&credArgs.RefreshToken)
	cmd.Flag("steam-account-name", "Steam account name that goes with --steam-refresh-token. Also read from "+steam.EnvAccountName+".").StringVar(&credArgs.AccountName)
	cmd.Flag("steam-publisher-key", "Steam publisher Web API key to use instead of the stored one. Also read from "+steam.EnvPublisherKey+".").StringVar(&credArgs.PublisherKey)
}

func store(ctx *mansion.Context) steam.Store {
	s := steam.StoreFor(ctx.Identity)
	s.Override = &credArgs
	return s
}

// hint turns the library's sentinel errors into the command to run.
func hint(err error) error {
	switch {
	case errors.Is(err, steam.ErrNotLoggedIn):
		return errors.New("not logged in to Steam, run `butler steam-login` first")
	case errors.Is(err, steam.ErrNoPublisherKey):
		return errors.New("no Steam publisher key stored, run `butler steam-key` first")
	}
	return err
}

func openSession(ctx *mansion.Context, goCtx context.Context) (*session.Session, error) {
	s, err := store(ctx).OpenSession(goCtx, comm.Debugf)
	if err != nil {
		return nil, hint(err)
	}
	return s, nil
}

func checkAppAccess(ctx *mansion.Context, goCtx context.Context, appID uint32) error {
	warnUngated()
	return hint(store(ctx).CheckAppAccess(goCtx, appID))
}

func warnUngated() {
	if steam.Ungated() {
		comm.Warnf("%s is set: the publisher key is not checked and any owned app can be synced. Development only.", steam.EnvUngated)
	}
}

func prompt(label string, secret bool) (string, error) {
	fmt.Fprint(os.Stderr, label)
	if secret && term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		return string(b), err
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
