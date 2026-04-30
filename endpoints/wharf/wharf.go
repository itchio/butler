package wharf

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
)

func Register(router *butlerd.Router) {
	messages.WharfPush.Register(router, Push)
	messages.WharfPushPreview.Register(router, PushPreview)
	messages.WharfListChannels.Register(router, ListChannels)
	messages.WharfGetChannel.Register(router, GetChannel)
	messages.WharfGetBuild.Register(router, GetBuild)
	messages.WharfListBuilds.Register(router, ListBuilds)
}
