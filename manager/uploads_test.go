package manager_test

import (
	"testing"

	"github.com/itchio/ox"
	"github.com/itchio/wharf/state"

	"github.com/itchio/butler/manager"
	itchio "github.com/itchio/go-itchio"
	"github.com/stretchr/testify/assert"
)

func makeTestConsumer(t *testing.T) *state.Consumer {
	return &state.Consumer{
		OnMessage: func(lvl string, msg string) {
			t.Helper()
			t.Logf("[%s] %s", lvl, msg)
		},
	}
}

func Test_NarrowDownUploads_FormatBlacklist(t *testing.T) {
	consumer := makeTestConsumer(t)

	game := &itchio.Game{
		Classification: itchio.GameClassificationGame,
	}

	ndu := func(uploads []*itchio.Upload, runtime *ox.Runtime) *manager.NarrowDownUploadsResult {
		return manager.NarrowDownUploads(consumer, game, uploads, runtime)
	}

	debrpm := []*itchio.Upload{
		{
			Platforms: itchio.Platforms{Linux: "all"},
			Filename:  "wrong.deb",
			Type:      "default",
		},
		{
			Platforms: itchio.Platforms{Linux: "all"},
			Filename:  "nope.rpm",
			Type:      "default",
		},
	}

	linux64 := &ox.Runtime{
		Platform: ox.PlatformLinux,
		Is64:     true,
	}

	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		HadWrongFormat: true,
		HadWrongArch:   false,
		Uploads:        nil,
		InitialUploads: debrpm,
	}, ndu(debrpm, linux64), "blacklist .deb and .rpm files")
}

func Test_NarrowDownUploads(t *testing.T) {
	consumer := makeTestConsumer(t)

	game := &itchio.Game{
		Classification: itchio.GameClassificationGame,
	}

	ndu := func(uploads []*itchio.Upload, runtime *ox.Runtime) *manager.NarrowDownUploadsResult {
		return manager.NarrowDownUploads(consumer, game, uploads, runtime)
	}

	linux64 := &ox.Runtime{
		Platform: ox.PlatformLinux,
		Is64:     true,
	}

	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		HadWrongFormat: false,
		HadWrongArch:   false,
		Uploads:        nil,
		InitialUploads: nil,
	}, ndu(nil, linux64), "empty is empty")

	mac64 := &ox.Runtime{
		Platform: ox.PlatformOSX,
		Is64:     true,
	}

	blacklistpkg := []*itchio.Upload{
		{
			Platforms: itchio.Platforms{OSX: "all"},
			Filename:  "super-mac-game.pkg",
			Type:      "default",
		},
	}
	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		HadWrongFormat: true,
		HadWrongArch:   false,
		Uploads:        nil,
		InitialUploads: blacklistpkg,
	}, ndu(blacklistpkg, mac64), "blacklist .pkg files")

	love := &itchio.Upload{
		Platforms: itchio.Platforms{OSX: "all", Linux: "all", Windows: "all"},
		Filename:  "no-really-all-platforms.love",
		Type:      "default",
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
	}, ndu(excludeuntagged, linux64), "exclude untagged, flag it")

	sources := &itchio.Upload{
		Platforms: itchio.Platforms{OSX: "all", Linux: "all", Windows: "all"},
		Filename:  "sources.tar.gz",
		Type:      "default",
	}

	linuxBinary := &itchio.Upload{
		Platforms: itchio.Platforms{Linux: "all"},
		Filename:  "binary.zip",
		Type:      "default",
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
	}, ndu(preferlinuxbin, linux64), "prefer linux binary")

	windowsNaked := &itchio.Upload{
		Platforms: itchio.Platforms{Windows: "all"},
		Filename:  "gamemaker-stuff-probably.exe",
		Type:      "default",
	}

	windowsPortable := &itchio.Upload{
		Platforms: itchio.Platforms{Windows: "all"},
		Filename:  "portable-build.zip",
		Type:      "default",
	}

	windows32 := &ox.Runtime{
		Platform: ox.PlatformWindows,
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
	}, ndu(preferwinportable, windows32), "prefer windows portable, then naked")

	windowsDemo := &itchio.Upload{
		Platforms: itchio.Platforms{Windows: "all"},
		Demo:      true,
		Filename:  "windows-demo.zip",
		Type:      "default",
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
	}, ndu(penalizedemos, windows32), "penalize demos")

	windows64 := &ox.Runtime{
		Platform: ox.PlatformWindows,
		Is64:     true,
	}

	loveWin := &itchio.Upload{
		Platforms: itchio.Platforms{Windows: "all"},
		Filename:  "win32.zip",
		Type:      "default",
	}

	loveMac := &itchio.Upload{
		Platforms: itchio.Platforms{OSX: "all"},
		Filename:  "mac64.zip",
		Type:      "default",
	}

	loveAll := &itchio.Upload{
		Platforms: itchio.Platforms{Windows: "all", Linux: "all", OSX: "all"},
		Filename:  "universal.zip",
		Type:      "default",
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
	}, ndu(preferexclusive, windows64), "prefer builds exclusive to platform")

	universalUpload := &itchio.Upload{
		Platforms: itchio.Platforms{Linux: "all"},
		Filename:  "Linux 32+64bit.tar.bz2",
		Type:      "default",
	}
	dontExcludeUniversal := []*itchio.Upload{
		universalUpload,
	}
	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		InitialUploads: dontExcludeUniversal,
		Uploads:        dontExcludeUniversal,
	}, ndu(dontExcludeUniversal, linux64), "don't exclude universal builds on 64-bit")

	linux32 := &ox.Runtime{
		Platform: ox.PlatformLinux,
		Is64:     false,
	}
	assert.EqualValues(t, &manager.NarrowDownUploadsResult{
		InitialUploads: dontExcludeUniversal,
		Uploads:        dontExcludeUniversal,
	}, ndu(dontExcludeUniversal, linux32), "don't exclude universal builds on 32-bit")

	{
		linux32Upload := &itchio.Upload{
			Platforms: itchio.Platforms{Linux: "386"},
			Filename:  "linux-386.tar.bz2",
			Type:      "default",
		}
		linux64Upload := &itchio.Upload{
			Platforms: itchio.Platforms{Linux: "amd64"},
			Filename:  "linux-amd64.tar.bz2",
			Type:      "default",
		}

		bothLinuxUploads := []*itchio.Upload{
			linux32Upload,
			linux64Upload,
		}

		assert.EqualValues(t, &manager.NarrowDownUploadsResult{
			InitialUploads: bothLinuxUploads,
			Uploads:        []*itchio.Upload{linux64Upload},
			HadWrongArch:   true,
		}, ndu(bothLinuxUploads, linux64), "do exclude 32-bit on 64-bit linux, if we have both")

		assert.EqualValues(t, &manager.NarrowDownUploadsResult{
			InitialUploads: bothLinuxUploads,
			Uploads:        []*itchio.Upload{linux32Upload},
			HadWrongArch:   true,
		}, ndu(bothLinuxUploads, linux32), "do exclude 64-bit on 32-bit linux, if we have both")
	}

	{
		windows32Upload := &itchio.Upload{
			Platforms: itchio.Platforms{Windows: "386"},
			Filename:  "Super Duper UE4 Game x86.rar",
			Type:      "default",
		}
		windows64Upload := &itchio.Upload{
			Platforms: itchio.Platforms{Windows: "amd64"},
			Filename:  "Super Duper UE4 Game x64.rar",
			Type:      "default",
		}

		bothWindowsUploads := []*itchio.Upload{
			windows32Upload,
			windows64Upload,
		}

		assert.EqualValues(t, &manager.NarrowDownUploadsResult{
			InitialUploads: bothWindowsUploads,
			Uploads:        []*itchio.Upload{windows64Upload},
			HadWrongArch:   true,
		}, ndu(bothWindowsUploads, windows64), "do exclude 32-bit on 64-bit windows, if we have both")

		assert.EqualValues(t, &manager.NarrowDownUploadsResult{
			InitialUploads: bothWindowsUploads,
			Uploads:        []*itchio.Upload{windows32Upload},
			HadWrongArch:   true,
		}, ndu(bothWindowsUploads, windows32), "do exclude 64-bit on 32-bit windows, if we have both")
	}
}
