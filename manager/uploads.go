package manager

import (
	"regexp"
	"sort"
	"strings"

	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/comm"
	itchio "github.com/itchio/go-itchio"
)

type NarrowDownUploadsResult struct {
	InitialUploads []*itchio.Upload
	Uploads        []*itchio.Upload
	HadWrongFormat bool
	HadWrongArch   bool
}

func NarrowDownUploads(uploads []*itchio.Upload, game *itchio.Game, runtime *Runtime) *NarrowDownUploadsResult {
	if actionForGame(game) == "open" {
		// we don't need any filtering for "open" action
		return &NarrowDownUploadsResult{
			InitialUploads: uploads,
			Uploads:        uploads,
			HadWrongFormat: false,
			HadWrongArch:   false,
		}
	}

	platformUploads := excludeWrongPlatform(uploads, runtime)
	formatUploads := excludeWrongFormat(platformUploads)
	hadWrongFormat := len(formatUploads) < len(platformUploads)

	archUploads := excludeWrongArch(formatUploads, runtime)
	hadWrongArch := len(archUploads) < len(formatUploads)

	sortedUploads := sortUploads(archUploads, runtime)

	return &NarrowDownUploadsResult{
		InitialUploads: uploads,
		Uploads:        sortedUploads,
		HadWrongFormat: hadWrongFormat,
		HadWrongArch:   hadWrongArch,
	}
}

func excludeWrongPlatform(uploads []*itchio.Upload, runtime *Runtime) []*itchio.Upload {
	var res []*itchio.Upload

	for _, u := range uploads {
		p := PlatformsForUpload(u)

		switch u.Type {
		case "default":
			if !p.IsCompatible(runtime) {
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

func excludeWrongFormat(uploads []*itchio.Upload) []*itchio.Upload {
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
	preferredFormatRegexp     = regexp.MustCompile(`\.(zip|7z)$`)
	usuallySourceFormatRegexp = regexp.MustCompile(`\.tar\.(gz|bz2|xz)$`)
	soundtrackFormatRegexp    = regexp.MustCompile(`soundtrack`)
)

func scoreUpload(upload *itchio.Upload, runtime *Runtime) *scoredUpload {
	filename := strings.ToLower(upload.Filename)
	var score int64 = 500

	if preferredFormatRegexp.MatchString(filename) {
		// Preferred formats
		score += 100
	} else if usuallySourceFormatRegexp.MatchString(filename) {
		// Usually not what you want (usually set of sources on Linux)
		score -= 100
	}

	// Definitely not something we can launch
	// Note: itch.io now has an upload type for soundtracks
	if soundtrackFormatRegexp.MatchString(filename) {
		score -= 1000
	}

	// Native uploads are preferred
	if upload.Type == "default" {
		score += 400
	}

	// Demos are penalized (if we have access to non-demo files)
	if upload.Demo {
		score -= 500
	}

	p := PlatformsForUpload(upload)
	score += p.ExclusivityScore(runtime)

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

func sortUploads(uploads []*itchio.Upload, runtime *Runtime) []*itchio.Upload {
	var scoredUploads []*scoredUpload

	for _, u := range uploads {
		scoredUploads = append(scoredUploads, scoreUpload(u, runtime))
	}

	sort.Stable(&highestScoreFirst{scoredUploads})

	var res []*itchio.Upload
	for _, su := range scoredUploads {
		res = append(res, su.upload)
	}

	return res
}

func excludeWrongArch(uploads []*itchio.Upload, runtime *Runtime) []*itchio.Upload {
	var res []*itchio.Upload

	filtered := false

	if runtime.Platform == buse.ItchPlatformWindows || runtime.Platform == buse.ItchPlatformLinux {
		comm.Logf("Got %d uploads, we're on %s, let's sniff architectures", len(uploads), runtime)

		if runtime.Is64 {
			// on 64-bit, if we have 64-bit builds, exclude 32-bit builds
			if anyUploadContainsString(uploads, "64") {
				filtered = true
				for _, u := range uploads {
					if uploadContainsString(u, "32") {
						// exclude
						continue
					}

					res = append(res, u)
				}
			}
		} else {
			// on 32-bit, if we have 32-bit builds, exclude 64-bit builds
			if anyUploadContainsString(uploads, "32") {
				for _, u := range uploads {
					if uploadContainsString(u, "64") {
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

func uploadContainsString(upload *itchio.Upload, s string) bool {
	return strings.Contains(strings.ToLower(upload.Filename), s) ||
		strings.Contains(strings.ToLower(upload.DisplayName), s)
}

func anyUploadContainsString(uploads []*itchio.Upload, s string) bool {
	for _, u := range uploads {
		if uploadContainsString(u, s) {
			return true
		}
	}

	return false
}
