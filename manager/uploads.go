package manager

import (
	"regexp"
	"sort"
	"strings"

	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/ox"
	"github.com/itchio/wharf/state"
)

type uploadFilter struct {
	consumer *state.Consumer
	runtime  *ox.Runtime
	game     *itchio.Game
}

type NarrowDownUploadsResult struct {
	InitialUploads []*itchio.Upload
	Uploads        []*itchio.Upload
	HadWrongFormat bool
	HadWrongArch   bool
}

func NarrowDownUploads(consumer *state.Consumer, game *itchio.Game, uploads []*itchio.Upload, runtime *ox.Runtime) *NarrowDownUploadsResult {
	uf := &uploadFilter{
		consumer: consumer,
		runtime:  runtime,
		game:     game,
	}

	return uf.narrowDownUploads(uploads)
}

func (uf *uploadFilter) narrowDownUploads(uploads []*itchio.Upload) *NarrowDownUploadsResult {
	platformUploads := uf.excludeWrongPlatform(uploads)
	formatUploads := uf.excludeWrongFormat(platformUploads)
	hadWrongFormat := len(formatUploads) < len(platformUploads)

	archUploads := uf.excludeWrongArch(formatUploads)
	hadWrongArch := len(archUploads) < len(formatUploads)

	sortedUploads := uf.sortUploads(archUploads)

	return &NarrowDownUploadsResult{
		InitialUploads: uploads,
		Uploads:        sortedUploads,
		HadWrongFormat: hadWrongFormat,
		HadWrongArch:   hadWrongArch,
	}
}

func (uf *uploadFilter) excludeWrongPlatform(uploads []*itchio.Upload) []*itchio.Upload {
	switch uf.game.Classification {
	case itchio.GameClassificationGame, itchio.GameClassificationTool:
		// apply regular filters
	default:
		// don't filter anything, cf. https://github.com/itchio/itch/issues/1958
		return uploads
	}

	var res []*itchio.Upload

	for _, u := range uploads {
		switch u.Type {
		case "default":
			if !IsCompatible(u.Platforms, uf.runtime) {
				// executable and not compatible with us? that's a skip
				continue
			}
		default:
			// soundtracks, books etc. - that's all good
		}

		res = append(res, u)
	}

	return res
}

var knownBadFormatRegexp = regexp.MustCompile(`(?i)\.(rpm|deb|pkg)$`)

func (uf *uploadFilter) excludeWrongFormat(uploads []*itchio.Upload) []*itchio.Upload {
	var res []*itchio.Upload

	for _, u := range uploads {
		if knownBadFormatRegexp.MatchString(u.Filename) {
			// package managers that don't have a silent flow are bad, sorry :(
			continue
		}

		res = append(res, u)
	}

	return res
}

type scoredUpload struct {
	score  int64
	upload *itchio.Upload
}

var (
	preferredFormatRegexp     = regexp.MustCompile(`\.(zip)$`)
	usuallySourceFormatRegexp = regexp.MustCompile(`\.tar\.(gz|bz2|xz)$`)
)

func (uf *uploadFilter) scoreUpload(upload *itchio.Upload) *scoredUpload {
	filename := strings.ToLower(upload.Filename)
	var score int64 = 500

	if preferredFormatRegexp.MatchString(filename) {
		// Preferred formats
		score += 100
	} else if usuallySourceFormatRegexp.MatchString(filename) {
		// Usually not what you want (usually set of sources on Linux)
		score -= 100
	}

	// We prefer things we can launch
	if upload.Type == "default" {
		score += 400
	}

	// Demos are penalized (if we have access to non-demo files)
	if upload.Demo {
		score -= 500
	}

	score += ExclusivityScore(upload.Platforms, uf.runtime)

	return &scoredUpload{
		score:  score,
		upload: upload,
	}
}

type highestScoreFirst struct {
	els []*scoredUpload
}

var _ sort.Interface = (*highestScoreFirst)(nil)

func (hsf *highestScoreFirst) Len() int {
	return len(hsf.els)
}

func (hsf *highestScoreFirst) Less(i, j int) bool {
	return hsf.els[i].score > hsf.els[j].score
}

func (hsf *highestScoreFirst) Swap(i, j int) {
	hsf.els[i], hsf.els[j] = hsf.els[j], hsf.els[i]
}

func (uf *uploadFilter) sortUploads(uploads []*itchio.Upload) []*itchio.Upload {
	var scoredUploads []*scoredUpload

	for _, u := range uploads {
		scoredUploads = append(scoredUploads, uf.scoreUpload(u))
	}

	sort.Stable(&highestScoreFirst{scoredUploads})

	var res []*itchio.Upload
	for _, su := range scoredUploads {
		res = append(res, su.upload)
	}

	return res
}

func (uf *uploadFilter) excludeWrongArch(uploads []*itchio.Upload) []*itchio.Upload {
	switch uf.runtime.Platform {
	case ox.PlatformWindows:
		if uf.runtime.Is64 {
			// on windows 64-bit, if we have both archs, exclude 32-bit builds
			if hasUploadsMatching(uploads, uploadIsWin64) {
				return excludeUploads(uploads, uploadIsWin32)
			}
		} else {
			// on windows 32-bit, if we have 32-bit builds, exclude 64-bit builds
			if hasUploadsMatching(uploads, uploadIsWin32) {
				return excludeUploads(uploads, uploadIsWin64)
			}
		}

	case ox.PlatformLinux:
		if uf.runtime.Is64 {
			// on 64-bit, if we have 64-bit builds, exclude 32-bit builds
			if hasUploadsMatching(uploads, uploadIsLinux64) {
				return excludeUploads(uploads, uploadIsLinux32)
			}
		} else {
			// on 32-bit, if we have 32-bit builds, exclude 64-bit builds
			if hasUploadsMatching(uploads, uploadIsLinux32) {
				return excludeUploads(uploads, uploadIsLinux64)
			}
		}
	}

	return uploads
}

func uploadIsLinux32(upload *itchio.Upload) bool {
	return upload.Platforms.Linux == itchio.Architectures386
}

func uploadIsLinux64(upload *itchio.Upload) bool {
	return upload.Platforms.Linux == itchio.ArchitecturesAmd64
}

func uploadIsWin32(upload *itchio.Upload) bool {
	return upload.Platforms.Windows == itchio.Architectures386
}

func uploadIsWin64(upload *itchio.Upload) bool {
	return upload.Platforms.Windows == itchio.ArchitecturesAmd64
}

func excludeUploads(uploads []*itchio.Upload, f func(u *itchio.Upload) bool) []*itchio.Upload {
	var res []*itchio.Upload
	for _, u := range uploads {
		if f(u) {
			continue
		}
		res = append(res, u)
	}
	return res
}

func hasUploadsMatching(uploads []*itchio.Upload, f func(u *itchio.Upload) bool) bool {
	for _, u := range uploads {
		if f(u) {
			return true
		}
	}

	return false
}
