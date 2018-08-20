package install

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
)

func Register(router *butlerd.Router) {
	messages.GameFindUploads.Register(router, GameFindUploads)
	messages.InstallQueue.Register(router, InstallQueue)
	messages.InstallPerform.Register(router, InstallPerform)
	messages.InstallCancel.Register(router, InstallCancel)
	messages.UninstallPerform.Register(router, UninstallPerform)
	messages.InstallVersionSwitchQueue.Register(router, InstallVersionSwitchQueue)
	messages.InstallLocationsGetByID.Register(router, InstallLocationsGetByID)
	messages.InstallLocationsList.Register(router, InstallLocationsList)
	messages.InstallLocationsAdd.Register(router, InstallLocationsAdd)
	messages.InstallLocationsRemove.Register(router, InstallLocationsRemove)
	messages.InstallLocationsScan.Register(router, InstallLocationsScan)

	messages.CavesSetPinned.Register(router, CavesSetPinned)
}
