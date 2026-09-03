package update

import (
	"testing"

	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/ox"
	"github.com/stretchr/testify/assert"
)

func Test_excludeForeignPlatform(t *testing.T) {
	linuxRT := ox.Runtime{Platform: ox.PlatformLinux, Is64: true}
	windowsRT := ox.Runtime{Platform: ox.PlatformWindows, Is64: true}

	linux := &itchio.Upload{ID: 1, Type: "default", Platforms: itchio.Platforms{Linux: itchio.ArchitecturesAll}}
	windows := &itchio.Upload{ID: 2, Type: "default", Platforms: itchio.Platforms{Windows: itchio.ArchitecturesAll}}
	both := &itchio.Upload{ID: 3, Type: "default", Platforms: itchio.Platforms{Linux: itchio.ArchitecturesAll, Windows: itchio.ArchitecturesAll}}
	soundtrack := &itchio.Upload{ID: 4, Type: "soundtrack"}
	unknown := &itchio.Upload{ID: 5, Type: "default"}

	candidates := []*itchio.Upload{linux, windows, both, soundtrack, unknown}

	// a linux install (possibly with wine available) must not be "updated" to a windows-only upload
	assert.Equal(t,
		[]*itchio.Upload{linux, both, soundtrack, unknown},
		excludeForeignPlatform(linux, candidates, linuxRT),
	)

	// a linux+windows upload runs natively on linux, so it must stay on linux too
	assert.Equal(t,
		[]*itchio.Upload{linux, both, soundtrack, unknown},
		excludeForeignPlatform(both, candidates, linuxRT),
	)

	assert.Equal(t,
		[]*itchio.Upload{windows, both, soundtrack, unknown},
		excludeForeignPlatform(both, candidates, windowsRT),
	)

	// a wine-wrapped windows install on linux still updates to windows uploads
	assert.Equal(t,
		[]*itchio.Upload{windows, both, soundtrack, unknown},
		excludeForeignPlatform(windows, candidates, linuxRT),
	)

	assert.Empty(t, excludeForeignPlatform(linux, []*itchio.Upload{windows}, linuxRT))

	// unknown installed platform: filtering would only guess
	assert.Equal(t, candidates, excludeForeignPlatform(nil, candidates, linuxRT))
	assert.Equal(t, candidates, excludeForeignPlatform(&itchio.Upload{ID: 9}, candidates, linuxRT))
	assert.Equal(t, candidates, excludeForeignPlatform(unknown, candidates, linuxRT))
	assert.Equal(t, candidates, excludeForeignPlatform(soundtrack, candidates, linuxRT))
}
