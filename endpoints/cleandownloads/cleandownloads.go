package cleandownloads

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/httpkit/progress"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/sizeof"
	"github.com/itchio/butler/cmd/wipe"
)

func Register(router *butlerd.Router) {
	messages.CleanDownloadsSearch.Register(router, CleanDownloadsSearch)
	messages.CleanDownloadsApply.Register(router, CleanDownloadsApply)
}

func CleanDownloadsSearch(rc *butlerd.RequestContext, params *butlerd.CleanDownloadsSearchParams) (*butlerd.CleanDownloadsSearchResult, error) {
	consumer := rc.Consumer

	// struct{} trick to use map as a set with 0-sized values
	whitemap := make(map[string]struct{})
	for _, whitelistPath := range params.Whitelist {
		whitemap[whitelistPath] = struct{}{}
	}

	var entries []*butlerd.CleanDownloadsEntry

	for _, root := range params.Roots {
		folders, err := ioutil.ReadDir(root)
		if err != nil {
			if os.IsNotExist(err) {
				// good, nothing to clean!
				continue
			} else {
				consumer.Warnf("Cannot scan root (%s): %s", root, err.Error())
				continue
			}
		}

		for _, folder := range folders {
			base := filepath.Base(folder.Name())
			absoluteFolderPath := filepath.Join(root, base)

			if _, ok := whitemap[base]; ok {
				// don't even consider it
				consumer.Debugf("Ignoring whitelisted (%s)")
				continue
			}

			// ey that's a candidate!
			folderSize, err := sizeof.Do(absoluteFolderPath)
			if err != nil {
				consumer.Warnf("Could not determine folder size: %s", err.Error())
			}

			entries = append(entries, &butlerd.CleanDownloadsEntry{
				Path: absoluteFolderPath,
				Size: folderSize,
			})
		}
	}

	res := &butlerd.CleanDownloadsSearchResult{
		Entries: entries,
	}
	return res, nil
}

func CleanDownloadsApply(rc *butlerd.RequestContext, params *butlerd.CleanDownloadsApplyParams) (*butlerd.CleanDownloadsApplyResult, error) {
	consumer := rc.Consumer

	for _, entry := range params.Entries {
		consumer.Infof("Wiping (%s) - %s", entry.Path, progress.FormatBytes(entry.Size))
		err := wipe.Do(consumer, entry.Path)
		if err != nil {
			consumer.Warnf("Could not wipe (%s): %s", entry.Path, err.Error())
		}
	}

	res := &butlerd.CleanDownloadsApplyResult{}
	return res, nil
}
