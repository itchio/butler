package install

import (
	"fmt"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/install/shortcut"
)

func InstallCreateShortcut(rc *butlerd.RequestContext, params butlerd.InstallCreateShortcutParams) (*butlerd.InstallCreateShortcutResult, error) {
	var cave *models.Cave

	rc.WithConn(func(conn *sqlite.Conn) {
		cave = models.CaveByID(conn, params.CaveID)
		models.PreloadCaves(conn, cave)
	})
	url := fmt.Sprintf("itch://caves/%s/launch", cave.ID)

	err := shortcut.Create(shortcut.CreateParams{
		DisplayName: cave.Game.Title,
		URL:         url,
		Consumer:    rc.Consumer,
	})
	if err != nil {
		return nil, err
	}

	res := &butlerd.InstallCreateShortcutResult{}
	return res, nil
}
