package downloads

import (
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
)

func Register(router *buse.Router) {
	messages.DownloadsQueue.Register(router, DownloadsQueue)
	messages.DownloadsPrioritize.Register(router, DownloadsPrioritize)
	messages.DownloadsList.Register(router, DownloadsList)
	messages.DownloadsDrive.Register(router, DownloadsDrive)
	messages.DownloadsClearFinished.Register(router, DownloadsClearFinished)
}
