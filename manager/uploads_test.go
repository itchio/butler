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
		InitialUploads: nil,
	}, manager.NarrowDownUploads(nil, game, linux64), "empty is empty")

	debrpm := []*itchio.Upload{
		&itchio.Upload{
			Linux:    true,
			Filename: "wrong.deb",
		},
		&itchio.Upload{
			Linux:    true,
			Filename: "nope.rpm",
		},
	}
	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		HadUntagged:    false,
		HadWrongFormat: true,
		HadWrongArch:   false,
		Uploads:        nil,
		InitialUploads: debrpm,
	}, manager.NarrowDownUploads(debrpm, game, linux64), "blacklist .deb and .rpm files")

	mac64 := &manager.Runtime{
		Platform: manager.ItchPlatformOSX,
		Is64:     true,
	}

	blacklistpkg := []*itchio.Upload{
		&itchio.Upload{
			OSX:      true,
			Filename: "super-mac-game.pkg",
		},
	}
	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		HadUntagged:    false,
		HadWrongFormat: true,
		HadWrongArch:   false,
		Uploads:        nil,
		InitialUploads: blacklistpkg,
	}, manager.NarrowDownUploads(blacklistpkg, game, mac64), "blacklist .pkg files")

	love := &itchio.Upload{
		Linux:    true,
		Windows:  true,
		OSX:      true,
		Filename: "no-really-all-platforms.love",
	}

	excludeuntagged := []*itchio.Upload{
		love,
		&itchio.Upload{
			Filename: "untagged-all-platforms.zip",
		},
	}
	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		InitialUploads: excludeuntagged,
		Uploads:        []*itchio.Upload{love},
		HadUntagged:    true,
		HadWrongFormat: false,
		HadWrongArch:   false,
	}, manager.NarrowDownUploads(excludeuntagged, game, linux64), "exclude untagged, flag it")

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

	preferlinuxbin := []*itchio.Upload{
		linuxBinary,
		sources,
		html,
	}
	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		InitialUploads: preferlinuxbin,
		Uploads: []*itchio.Upload{
			linuxBinary,
			sources,
			html,
		},
		HadUntagged:    false,
		HadWrongFormat: false,
		HadWrongArch:   false,
	}, manager.NarrowDownUploads(preferlinuxbin, game, linux64), "prefer linux binary")

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

	preferwinportable := []*itchio.Upload{
		html,
		windowsPortable,
		windowsNaked,
	}
	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		InitialUploads: preferwinportable,
		Uploads: []*itchio.Upload{
			windowsPortable,
			windowsNaked,
			html,
		},
		HadUntagged:    false,
		HadWrongFormat: false,
		HadWrongArch:   false,
	}, manager.NarrowDownUploads(preferwinportable, game, windows32), "prefer windows portable, then naked")

	windowsDemo := &itchio.Upload{
		Windows:  true,
		Demo:     true,
		Filename: "windows-demo.zip",
	}

	penalizedemos := []*itchio.Upload{
		windowsDemo,
		windowsPortable,
		windowsNaked,
	}
	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		InitialUploads: penalizedemos,
		Uploads: []*itchio.Upload{
			windowsPortable,
			windowsNaked,
			windowsDemo,
		},
		HadUntagged:    false,
		HadWrongFormat: false,
		HadWrongArch:   false,
	}, manager.NarrowDownUploads(penalizedemos, game, windows32), "penalize demos")

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

	preferexclusive := []*itchio.Upload{
		loveAll,
		loveWin,
		loveMac,
	}
	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		InitialUploads: preferexclusive,
		Uploads: []*itchio.Upload{
			loveWin,
			loveAll,
		},
		HadUntagged:    false,
		HadWrongFormat: false,
		HadWrongArch:   false,
	}, manager.NarrowDownUploads(preferexclusive, game, windows64), "prefer builds exclusive to platform")
}
