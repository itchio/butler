package wharf

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
)

func Register(router *butlerd.Router) {
	messages.WharfPush.Register(router, Push)
	messages.WharfListChannels.Register(router, ListChannels)
	messages.WharfGetChannel.Register(router, GetChannel)
	messages.WharfGetBuild.Register(router, GetBuild)
}
