package launch

import (
	"fmt"
	"os"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/manager/runlock"
	"github.com/pkg/errors"

	"github.com/itchio/ox"

	validation "github.com/go-ozzo/ozzo-validation"
)

type withInstallFolderLockParams struct {
	rc     *butlerd.RequestContext
	caveID string
	reason string
}

type withInstallFolderInfo struct {
	installFolder string
	cave          *models.Cave
	access        *operate.GameAccess
	runtime       *ox.Runtime
}

func withInstallFolderLock(params withInstallFolderLockParams, f func(info withInstallFolderInfo) error) error {
	err := validation.ValidateStruct(&params,
		validation.Field(&params.rc, validation.Required),
		validation.Field(&params.caveID, validation.Required),
		validation.Field(&params.reason, validation.Required),
	)
	if err != nil {
		return err
	}

	rc := params.rc
	consumer := rc.Consumer

	cave := operate.ValidateCave(rc, params.caveID)
	var installFolder string
	rc.WithConn(func(conn *sqlite.Conn) {
		installFolder = cave.GetInstallFolder(conn)
	})

	_, err = os.Stat(installFolder)
	if err != nil && os.IsNotExist(err) {
		return &butlerd.RpcError{
			Code:    int64(butlerd.CodeInstallFolderDisappeared),
			Message: fmt.Sprintf("Could not find install folder (%s)", installFolder),
		}
	}

	rlock := runlock.New(consumer, installFolder)
	err = rlock.Lock(rc.Ctx, params.reason)
	if err != nil {
		return errors.WithStack(err)
	}
	defer rlock.Unlock()

	var access *operate.GameAccess
	rc.WithConn(func(conn *sqlite.Conn) {
		access = operate.AccessForGameID(conn, cave.Game.ID).OnlyAPIKey()
	})

	runtime := ox.CurrentRuntime()

	info := withInstallFolderInfo{
		installFolder,
		cave,
		access,
		runtime,
	}

	return f(info)
}
