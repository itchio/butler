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
	game     *itchio.Game
	runtime  *ox.Runtime
}

type NarrowDownUploadsResult struct {
	InitialUploads []*itchio.Upload
	Uploads        []*itchio.Upload
	HadWrongFormat bool
	HadWrongArch   bool
}

func NarrowDownUploads(consumer *state.Consumer, uploads []*itchio.Upload, game *itchio.Game, runtime *ox.Runtime) *NarrowDownUploadsResult {
	uf := &uploadFilter{
		consumer: consumer,
		game:     game,
		runtime:  runtime,
	}

	return uf.narrowDownUploads(uploads)
}

func (uf *uploadFilter) narrowDownUploads(uploads []*itchio.Upload) *NarrowDownUploadsResult {
	if actionForGame(uf.game) == "open" {
		// we don't need any filtering for "open" action
		return &NarrowDownUploadsResult{
			InitialUploads: uploads,
			Uploads:        uploads,
			HadWrongFormat: false,
			HadWrongArch:   false,
		}
	}

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
	var res []*itchio.Upload

	for _, u := range uploads {
		p := PlatformsForUpload(u)

		switch u.Type {
		case "default":
			if !p.IsCompatible(uf.runtime) {
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
	if upload.Traits.Demo {
		score -= 500
	}

	p := PlatformsForUpload(upload)
	score += p.ExclusivityScore(uf.runtime)

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

// TODO: rely on server-side metadata instead
func (uf *uploadFilter) excludeWrongArch(uploads []*itchio.Upload) []*itchio.Upload {
	var res []*itchio.Upload

	filtered := false

	if uf.runtime.Platform == ox.PlatformWindows || uf.runtime.Platform == ox.PlatformLinux {
		uf.consumer.Logf("Got %d uploads, we're on %s, let's sniff architectures", len(uploads), uf.runtime)

		if uf.runtime.Is64 {
			// on 64-bit, if we have 64-bit builds, exclude 32-bit builds
			if hasUploadsMatching(uploads, uploadSeems64Bit) {
				filtered = true
				for _, u := range uploads {
					if uploadSeems32Bit(u) && !uploadSeems64Bit(u) {
						// exclude
						continue
					}

					res = append(res, u)
				}
			}
		} else {
			// on 32-bit, if we have 32-bit builds, exclude 64-bit builds
			if hasUploadsMatching(uploads, uploadSeems32Bit) {
				filtered = true
				for _, u := range uploads {
					if uploadSeems64Bit(u) && !uploadSeems32Bit(u) {
						// exclude
						continue
					}

					res = append(res, u)
				}
			}
		}
	}

	if filtered {
		return res
	}
	return uploads
}

func uploadSeems32Bit(upload *itchio.Upload) bool {
	return uploadContainsAnyString(upload, []string{"386", "686", "x86", "32"})
}

func uploadSeems64Bit(upload *itchio.Upload) bool {
	// covers amd64, x64, etc.
	return uploadContainsAnyString(upload, []string{"64"})
}

func uploadContainsAnyString(upload *itchio.Upload, queries []string) bool {
	lowerFileName := strings.ToLower(upload.Filename)
	lowerDisplayName := strings.ToLower(upload.DisplayName)

	for _, q := range queries {
		if strings.Contains(lowerFileName, q) {
			return true
		}
		if strings.Contains(lowerDisplayName, q) {
			return true
		}
	}
	return false
}

func hasUploadsMatching(uploads []*itchio.Upload, f func(u *itchio.Upload) bool) bool {
	for _, u := range uploads {
		if f(u) {
			return true
		}
	}

	return false
}
