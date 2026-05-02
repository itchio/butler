package publish

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
)

func Register(router *butlerd.Router) {
	messages.PublishPush.Register(router, Push)
	messages.PublishPushPreview.Register(router, PushPreview)
	messages.PublishListChannels.Register(router, ListChannels)
	messages.PublishGetChannel.Register(router, GetChannel)
	messages.PublishGetBuild.Register(router, GetBuild)
	messages.PublishListBuilds.Register(router, ListBuilds)
}
