package manager_test

import (
	"testing"

	"github.com/itchio/butler/manager"
	itchio "github.com/itchio/go-itchio"
	"github.com/stretchr/testify/assert"
)

func Test_NarrowDownUploads(t *testing.T) {
	game := &itchio.Game{
		ID:             123,
		Classification: "game",
	}

	linux64 := &manager.Runtime{
		Platform: manager.ItchPlatformLinux,
		Is64:     true,
	}

	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		HadUntagged:    false,
		HadWrongFormat: false,
		HadWrongArch:   false,
		Uploads:        nil,
	}, manager.NarrowDownUploads(nil, game, linux64), "empty is empty")

	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		HadUntagged:    false,
		HadWrongFormat: true,
		HadWrongArch:   false,
		Uploads:        nil,
	}, manager.NarrowDownUploads([]*itchio.Upload{
		&itchio.Upload{
			Linux:    true,
			Filename: "wrong.deb",
		},
		&itchio.Upload{
			Linux:    true,
			Filename: "nope.rpm",
		},
	}, game, linux64), "blacklist .deb and .rpm files")

	mac64 := &manager.Runtime{
		Platform: manager.ItchPlatformOSX,
		Is64:     true,
	}

	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		HadUntagged:    false,
		HadWrongFormat: true,
		HadWrongArch:   false,
		Uploads:        nil,
	}, manager.NarrowDownUploads([]*itchio.Upload{
		&itchio.Upload{
			OSX:      true,
			Filename: "super-mac-game.pkg",
		},
	}, game, mac64), "blacklist .pkg files")

	love := &itchio.Upload{
		Linux:    true,
		Windows:  true,
		OSX:      true,
		Filename: "no-really-all-platforms.love",
	}

	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		Uploads:        []*itchio.Upload{love},
		HadUntagged:    true,
		HadWrongFormat: false,
		HadWrongArch:   false,
	}, manager.NarrowDownUploads([]*itchio.Upload{
		love,
		&itchio.Upload{
			Filename: "untagged-all-platforms.zip",
		},
	}, game, linux64), "exclude untagged, flag it")

	sources := &itchio.Upload{
		Linux:    true,
		Windows:  true,
		OSX:      true,
		Filename: "sources.tar.gz",
	}

	linuxBinary := &itchio.Upload{
		Linux:    true,
		Filename: "binary.zip",
	}

	html := &itchio.Upload{
		Type:     "html",
		Filename: "twine-is-not-a-twemulator.zip",
	}

	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		Uploads: []*itchio.Upload{
			linuxBinary,
			sources,
			html,
		},
		HadUntagged:    false,
		HadWrongFormat: false,
		HadWrongArch:   false,
	}, manager.NarrowDownUploads([]*itchio.Upload{
		linuxBinary,
		sources,
		html,
	}, game, linux64), "prefer linux binary")

	windowsNaked := &itchio.Upload{
		Windows:  true,
		Filename: "gamemaker-stuff-probably.exe",
	}

	windowsPortable := &itchio.Upload{
		Windows:  true,
		Filename: "portable-build.zip",
	}

	windows32 := &manager.Runtime{
		Platform: manager.ItchPlatformWindows,
		Is64:     false,
	}

	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		Uploads: []*itchio.Upload{
			windowsPortable,
			windowsNaked,
			html,
		},
		HadUntagged:    false,
		HadWrongFormat: false,
		HadWrongArch:   false,
	}, manager.NarrowDownUploads([]*itchio.Upload{
		html,
		windowsPortable,
		windowsNaked,
	}, game, windows32), "prefer windows portable, then naked")

	windowsDemo := &itchio.Upload{
		Windows:  true,
		Demo:     true,
		Filename: "windows-demo.zip",
	}

	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		Uploads: []*itchio.Upload{
			windowsPortable,
			windowsNaked,
			windowsDemo,
		},
		HadUntagged:    false,
		HadWrongFormat: false,
		HadWrongArch:   false,
	}, manager.NarrowDownUploads([]*itchio.Upload{
		windowsDemo,
		windowsPortable,
		windowsNaked,
	}, game, windows32), "penalize demos")

	windows64 := &manager.Runtime{
		Platform: manager.ItchPlatformWindows,
		Is64:     true,
	}

	loveWin := &itchio.Upload{
		Windows:  true,
		Filename: "win32.zip",
	}

	loveMac := &itchio.Upload{
		OSX:      true,
		Filename: "mac64.zip",
	}

	loveAll := &itchio.Upload{
		Windows:  true,
		OSX:      true,
		Linux:    true,
		Filename: "universal.zip",
	}

	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		Uploads: []*itchio.Upload{
			loveWin,
			loveAll,
		},
		HadUntagged:    false,
		HadWrongFormat: false,
		HadWrongArch:   false,
	}, manager.NarrowDownUploads([]*itchio.Upload{
		loveAll,
		loveWin,
		loveMac,
	}, game, windows64), "prefer builds exclusive to platform")
}
