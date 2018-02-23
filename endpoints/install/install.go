package install

import (
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
)

func Register(router *buse.Router) {
	messages.GameFindUploads.Register(router, GameFindUploads)
	messages.OperationStart.Register(router, OperationStart)
	messages.OperationCancel.Register(router, OperationCancel)
}
