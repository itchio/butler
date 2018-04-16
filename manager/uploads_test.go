package manager_test

import (
	"testing"

	"github.com/itchio/wharf/state"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/manager"
	itchio "github.com/itchio/go-itchio"
	"github.com/stretchr/testify/assert"
)

func Test_NarrowDownUploads(t *testing.T) {
	consumer := &state.Consumer{
		OnMessage: func(lvl string, msg string) {
			t.Logf("[%s] %s", lvl, msg)
		},
	}

	game := &itchio.Game{
		ID:             123,
		Classification: "game",
	}

	linux64 := &manager.Runtime{
		Platform: butlerd.ItchPlatformLinux,
		Is64:     true,
	}

	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		HadWrongFormat: false,
		HadWrongArch:   false,
		Uploads:        nil,
		InitialUploads: nil,
	}, manager.NarrowDownUploads(consumer, nil, game, linux64), "empty is empty")

	debrpm := []*itchio.Upload{
		{
			Linux:    true,
			Filename: "wrong.deb",
			Type:     "default",
		},
		{
			Linux:    true,
			Filename: "nope.rpm",
			Type:     "default",
		},
	}
	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		HadWrongFormat: true,
		HadWrongArch:   false,
		Uploads:        nil,
		InitialUploads: debrpm,
	}, manager.NarrowDownUploads(consumer, debrpm, game, linux64), "blacklist .deb and .rpm files")

	mac64 := &manager.Runtime{
		Platform: butlerd.ItchPlatformOSX,
		Is64:     true,
	}

	blacklistpkg := []*itchio.Upload{
		{
			OSX:      true,
			Filename: "super-mac-game.pkg",
			Type:     "default",
		},
	}
	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		HadWrongFormat: true,
		HadWrongArch:   false,
		Uploads:        nil,
		InitialUploads: blacklistpkg,
	}, manager.NarrowDownUploads(consumer, blacklistpkg, game, mac64), "blacklist .pkg files")

	love := &itchio.Upload{
		Linux:    true,
		Windows:  true,
		OSX:      true,
		Filename: "no-really-all-platforms.love",
		Type:     "default",
	}

	excludeuntagged := []*itchio.Upload{
		love,
		{
			Filename: "untagged-all-platforms.zip",
			Type:     "default",
		},
	}
	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		InitialUploads: excludeuntagged,
		Uploads:        []*itchio.Upload{love},
		HadWrongFormat: false,
		HadWrongArch:   false,
	}, manager.NarrowDownUploads(consumer, excludeuntagged, game, linux64), "exclude untagged, flag it")

	sources := &itchio.Upload{
		Linux:    true,
		Windows:  true,
		OSX:      true,
		Filename: "sources.tar.gz",
		Type:     "default",
	}

	linuxBinary := &itchio.Upload{
		Linux:    true,
		Filename: "binary.zip",
		Type:     "default",
	}

	html := &itchio.Upload{
		Filename: "twine-is-not-a-twemulator.zip",
		Type:     "html",
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
		HadWrongFormat: false,
		HadWrongArch:   false,
	}, manager.NarrowDownUploads(consumer, preferlinuxbin, game, linux64), "prefer linux binary")

	windowsNaked := &itchio.Upload{
		Windows:  true,
		Filename: "gamemaker-stuff-probably.exe",
		Type:     "default",
	}

	windowsPortable := &itchio.Upload{
		Windows:  true,
		Filename: "portable-build.zip",
		Type:     "default",
	}

	windows32 := &manager.Runtime{
		Platform: butlerd.ItchPlatformWindows,
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
		HadWrongFormat: false,
		HadWrongArch:   false,
	}, manager.NarrowDownUploads(consumer, preferwinportable, game, windows32), "prefer windows portable, then naked")

	windowsDemo := &itchio.Upload{
		Windows:  true,
		Demo:     true,
		Filename: "windows-demo.zip",
		Type:     "default",
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
		HadWrongFormat: false,
		HadWrongArch:   false,
	}, manager.NarrowDownUploads(consumer, penalizedemos, game, windows32), "penalize demos")

	windows64 := &manager.Runtime{
		Platform: butlerd.ItchPlatformWindows,
		Is64:     true,
	}

	loveWin := &itchio.Upload{
		Windows:  true,
		Filename: "win32.zip",
		Type:     "default",
	}

	loveMac := &itchio.Upload{
		OSX:      true,
		Filename: "mac64.zip",
		Type:     "default",
	}

	loveAll := &itchio.Upload{
		Windows:  true,
		OSX:      true,
		Linux:    true,
		Filename: "universal.zip",
		Type:     "default",
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
		HadWrongFormat: false,
		HadWrongArch:   false,
	}, manager.NarrowDownUploads(consumer, preferexclusive, game, windows64), "prefer builds exclusive to platform")

	universalUpload := &itchio.Upload{
		Linux:    true,
		Filename: "Linux 32+64bit.tar.bz2",
		Type:     "default",
	}
	dontExcludeUniversal := []*itchio.Upload{
		universalUpload,
	}
	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		InitialUploads: dontExcludeUniversal,
		Uploads:        dontExcludeUniversal,
	}, manager.NarrowDownUploads(consumer, dontExcludeUniversal, game, linux64), "don't exclude universal builds on 64-bit")

	linux32 := &manager.Runtime{
		Platform: butlerd.ItchPlatformLinux,
		Is64:     false,
	}
	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		InitialUploads: dontExcludeUniversal,
		Uploads:        dontExcludeUniversal,
	}, manager.NarrowDownUploads(consumer, dontExcludeUniversal, game, linux32), "don't exclude universal builds on 32-bit")

	{
		linux32Upload := &itchio.Upload{
			Linux:    true,
			Filename: "linux-386.tar.bz2",
			Type:     "default",
		}
		linux64Upload := &itchio.Upload{
			Linux:    true,
			Filename: "linux-amd64.tar.bz2",
			Type:     "default",
		}

		bothLinuxUploads := []*itchio.Upload{
			linux32Upload,
			linux64Upload,
		}

		assert.EqualValues(t, &manager.NarrowDownUploadsResult{
			InitialUploads: bothLinuxUploads,
			Uploads:        []*itchio.Upload{linux64Upload},
			HadWrongArch:   true,
		}, manager.NarrowDownUploads(consumer, bothLinuxUploads, game, linux64), "do exclude 32-bit on 64-bit linux, if we have both")

		assert.EqualValues(t, &manager.NarrowDownUploadsResult{
			InitialUploads: bothLinuxUploads,
			Uploads:        []*itchio.Upload{linux32Upload},
			HadWrongArch:   true,
		}, manager.NarrowDownUploads(consumer, bothLinuxUploads, game, linux32), "do exclude 64-bit on 32-bit linux, if we have both")
	}

	{
		windows32Upload := &itchio.Upload{
			Windows:  true,
			Filename: "Super Duper UE4 Game x86.rar",
			Type:     "default",
		}
		windows64Upload := &itchio.Upload{
			Windows:  true,
			Filename: "Super Duper UE4 Game x64.rar",
			Type:     "default",
		}

		bothWindowsUploads := []*itchio.Upload{
			windows32Upload,
			windows64Upload,
		}

		assert.EqualValues(t, &manager.NarrowDownUploadsResult{
			InitialUploads: bothWindowsUploads,
			Uploads:        []*itchio.Upload{windows64Upload},
			HadWrongArch:   true,
		}, manager.NarrowDownUploads(consumer, bothWindowsUploads, game, windows64), "do exclude 32-bit on 64-bit windows, if we have both")

		assert.EqualValues(t, &manager.NarrowDownUploadsResult{
			InitialUploads: bothWindowsUploads,
			Uploads:        []*itchio.Upload{windows32Upload},
			HadWrongArch:   true,
		}, manager.NarrowDownUploads(consumer, bothWindowsUploads, game, windows32), "do exclude 64-bit on 32-bit windows, if we have both")
	}
}
