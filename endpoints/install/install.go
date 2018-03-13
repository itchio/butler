package install

import (
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
)

func Register(router *buse.Router) {
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
}
