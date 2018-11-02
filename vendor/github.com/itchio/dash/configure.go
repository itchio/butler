package dash

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/itchio/wharf/pools"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
)

func sniffPoolEntry(pool wsync.Pool, fileIndex int64, file *tlc.File) (*Candidate, error) {
	r, err := pool.GetReadSeeker(fileIndex)
	if err != nil {
		return nil, errors.Wrap(err, "while getting read seeker for pool entry")
	}

	size := pool.GetSize(fileIndex)

	return Sniff(r, file.Path, size)
}

func Sniff(r io.ReadSeeker, path string, size int64) (*Candidate, error) {
	c, err := doSniff(r, path, size)
	if c != nil {
		c.Size = size
		if c.Path == "" {
			c.Path = path
		}
		c.Depth = pathDepth(c.Path)
	}
	return c, err
}

func doSniff(r io.ReadSeeker, path string, size int64) (*Candidate, error) {
	lowerPath := strings.ToLower(path)

	lowerBase := filepath.Base(lowerPath)
	dir := filepath.Dir(path)
	switch lowerBase {
	case "index.html":
		return &Candidate{
			Flavor: FlavorHTML,
			Path:   path,
		}, nil
	case "conf.lua":
		return sniffLove(r, size, dir)
	}

	if strings.HasSuffix(lowerPath, ".love") {
		return &Candidate{
			Flavor: FlavorLove,
			Path:   path,
		}, nil
	}

	// if it ends in .exe, it's probably an .exe
	if strings.HasSuffix(lowerPath, ".exe") {
		subRes, subErr := sniffPE(r, size)
		if subErr != nil {
			return nil, errors.Wrap(subErr, "sniffing PE file")
		}
		if subRes != nil {
			// it was an exe!
			return subRes, nil
		}
		// it wasn't an exe, carry on...
	}

	// if it ends in .bat or .cmd, it's a windows script
	if strings.HasSuffix(lowerPath, ".bat") || strings.HasSuffix(lowerPath, ".cmd") {
		return &Candidate{
			Flavor: FlavorScriptWindows,
		}, nil
	}

	buf := make([]byte, 8)
	n, _ := io.ReadFull(r, buf)
	if n < len(buf) {
		// too short to be an exec or unreadable
		return nil, nil
	}

	// intel Mach-O executables start with 0xCEFAEDFE or 0xCFFAEDFE
	// (old PowerPC Mach-O executables started with 0xFEEDFACE)
	if (buf[0] == 0xCE || buf[0] == 0xCF) && buf[1] == 0xFA && buf[2] == 0xED && buf[3] == 0xFE {
		return &Candidate{
			Flavor: FlavorNativeMacos,
		}, nil
	}

	// Mach-O universal binaries start with 0xCAFEBABE
	// it's Apple's 'fat binary' stuff that contains multiple architectures
	// unfortunately, compiled Java classes also start with that
	if buf[0] == 0xCA && buf[1] == 0xFE && buf[2] == 0xBA && buf[3] == 0xBE {
		return sniffFatMach(r, size)
	}

	// ELF executables start with 0x7F454C46
	// (e.g. 0x7F + 'ELF' in ASCII)
	if buf[0] == 0x7F && buf[1] == 0x45 && buf[2] == 0x4C && buf[3] == 0x46 {
		return sniffELF(r, size)
	}

	// Shell scripts start with a shebang (#!)
	// https://en.wikipedia.org/wiki/Shebang_(Unix)
	if buf[0] == 0x23 && buf[1] == 0x21 {
		return sniffScript(r, size)
	}

	// MSI (Microsoft Installer Packages) have a well-defined magic number.
	if buf[0] == 0xD0 && buf[1] == 0xCF &&
		buf[2] == 0x11 && buf[3] == 0xE0 &&
		buf[4] == 0xA1 && buf[5] == 0xB1 &&
		buf[6] == 0x1A && buf[7] == 0xE1 {
		return &Candidate{
			Flavor: FlavorNativeWindows,
			WindowsInfo: &WindowsInfo{
				InstallerType: WindowsInstallerTypeMsi,
			},
		}, nil
	}

	if buf[0] == 0x50 && buf[1] == 0x4B &&
		buf[2] == 0x03 && buf[3] == 0x04 {
		return sniffZip(r, size)
	}

	return nil, nil
}

// ConfigureParams controls the behavior of Configure
type ConfigureParams struct {
	Consumer *state.Consumer
	Filter   tlc.FilterFunc
}

// Configure walks a directory and finds potential launch candidates,
// grouped together into a verdict.
func Configure(root string, params *ConfigureParams) (*Verdict, error) {
	if params == nil {
		return nil, errors.New("missing params")
	}
	consumer := params.Consumer

	filter := params.Filter
	if filter == nil {
		filter = func(fi os.FileInfo) bool {
			if fi.Name() == ".itch" {
				return false
			}
			return true
		}
	}

	verdict := &Verdict{
		BasePath: root,
	}

	var pool wsync.Pool

	container, err := tlc.WalkAny(root, &tlc.WalkOpts{Filter: filter})
	if err != nil {
		return nil, err
	}

	if pool == nil {
		pool, err = pools.New(container, root)
		if err != nil {
			return nil, errors.Wrap(err, "creating pool to configure folder")
		}
	}

	defer pool.Close()

	var candidates = make([]*Candidate, 0)

	for _, d := range container.Dirs {
		lowerPath := strings.ToLower(d.Path)
		if strings.HasSuffix(lowerPath, ".app") {
			plistPath := lowerPath + "/contents/info.plist"

			plistFound := false
			for _, f := range container.Files {
				if strings.ToLower(f.Path) == plistPath {
					plistFound = true
					break
				}
			}

			if !plistFound {
				consumer.Logf("Found app bundle without an Info.plist: %s", d.Path)
				continue
			}

			res := &Candidate{
				Flavor: FlavorAppMacos,
				Size:   0,
				Path:   d.Path,
				Mode:   d.Mode,
			}
			res.Depth = pathDepth(res.Path)
			candidates = append(candidates, res)
		}
	}

	for fileIndex, f := range container.Files {
		verdict.TotalSize += f.Size

		res, err := sniffPoolEntry(pool, int64(fileIndex), f)
		if err != nil {
			return nil, errors.Wrap(err, "sniffing pool entry")
		}

		if res != nil {
			res.Mode = f.Mode
			candidates = append(candidates, res)
		}
	}

	if len(candidates) == 0 && container.IsSingleFile() {
		f := container.Files[0]

		if hasExt(f.Path, ".html") {
			// ok, that's an HTML5 game
			candidate := &Candidate{
				Size:   f.Size,
				Path:   f.Path,
				Mode:   f.Mode,
				Depth:  pathDepth(f.Path),
				Flavor: FlavorHTML,
			}
			candidates = append(candidates, candidate)
		}
	}

	if len(candidates) == 0 {
		// still no candidates? if we have a top-level .html file, let's go for it
		for _, f := range container.Files {
			if pathDepth(f.Path) == 1 && hasExt(f.Path, ".html") {
				// ok, that's an HTML5 game
				candidate := &Candidate{
					Size:   f.Size,
					Path:   f.Path,
					Mode:   f.Mode,
					Depth:  pathDepth(f.Path),
					Flavor: FlavorHTML,
				}
				candidates = append(candidates, candidate)
			}
		}
	}

	verdict.Candidates = candidates

	return verdict, nil
}

type FixPermissionsParams struct {
	DryRun   bool
	Consumer *state.Consumer
}

// FixPermissions makes sure all ELF executables, COFF executables,
// and scripts have the executable bit set
func FixPermissions(v *Verdict, params *FixPermissionsParams) ([]string, error) {
	if params == nil {
		return nil, errors.New("missing params")
	}

	consumer := params.Consumer

	var fixed []string

	var libraryPattern = regexp.MustCompile(`\.so(\.[0-9]+)*$`)

	for _, c := range v.Candidates {
		switch c.Flavor {
		case FlavorNativeLinux, FlavorNativeMacos, FlavorScript:
			baseName := filepath.Base(c.Path)
			if libraryPattern.MatchString(baseName) {
				// don't fix dynamic libraries linux
				continue
			}

			fullPath := filepath.Join(v.BasePath, c.Path)

			if c.Mode&0100 == 0 {
				consumer.Logf("Fixing permissions for %s", c.Path)

				fixed = append(fixed, c.Path)
				if !params.DryRun {
					err := os.Chmod(fullPath, 0755)
					if err != nil {
						return nil, err
					}
				}
			}
		}

		c.Mode = 0
	}

	return fixed, nil
}

type biggestFirst struct {
	candidates []*Candidate
}

var _ sort.Interface = (*biggestFirst)(nil)

func (bf *biggestFirst) Len() int {
	return len(bf.candidates)
}

func (bf *biggestFirst) Less(i, j int) bool {
	return bf.candidates[i].Size > bf.candidates[j].Size
}

func (bf *biggestFirst) Swap(i, j int) {
	bf.candidates[i], bf.candidates[j] = bf.candidates[j], bf.candidates[i]
}

type HighestScoreFirst struct {
	candidates []ScoredCandidate
}

var _ sort.Interface = (*HighestScoreFirst)(nil)

func (hsf *HighestScoreFirst) Len() int {
	return len(hsf.candidates)
}

func (hsf *HighestScoreFirst) Less(i, j int) bool {
	return hsf.candidates[i].score > hsf.candidates[j].score
}

func (hsf *HighestScoreFirst) Swap(i, j int) {
	hsf.candidates[i], hsf.candidates[j] = hsf.candidates[j], hsf.candidates[i]
}

type BlacklistEntry struct {
	pattern *regexp.Regexp
	penalty Penalty
}

type PenaltyKind int

const (
	PenaltyExclude = iota
	PenaltyScore
)

type Penalty struct {
	kind  PenaltyKind
	delta int64
}

var blacklist = []BlacklistEntry{
	{regexp.MustCompile(`(?i)unins.*\.exe$`), Penalty{PenaltyScore, 50}},
	{regexp.MustCompile(`(?i)kick\.bin$`), Penalty{PenaltyScore, 50}},
	{regexp.MustCompile(`(?i)\.vshost\.exe$`), Penalty{PenaltyScore, 50}},
	{regexp.MustCompile(`(?i)nacl_helper`), Penalty{PenaltyScore, 20}},
	{regexp.MustCompile(`(?i)nwjc\.exe$`), Penalty{PenaltyScore, 20}},
	{regexp.MustCompile(`(?i)flixel\.exe$`), Penalty{PenaltyScore, 20}},
	{regexp.MustCompile(`(?i)\.(so|dylib)$`), Penalty{PenaltyExclude, 0}},
	{regexp.MustCompile(`(?i)dxwebsetup\.exe$`), Penalty{PenaltyExclude, 0}},
	{regexp.MustCompile(`(?i)vcredist.*\.exe$`), Penalty{PenaltyExclude, 0}},
	{regexp.MustCompile(`(?i)unitycrashhandler.*\.exe$`), Penalty{PenaltyExclude, 0}},
}

type ScoredCandidate struct {
	candidate *Candidate
	score     int64
}

func (v *Verdict) FilterPlatform(osFilter string, archFilter string) {
	var compatibleCandidates []*Candidate

	// exclude things we can't run at all
	for _, c := range v.Candidates {
		keep := true

		switch c.Flavor {
		case FlavorNativeLinux:
			if osFilter != "linux" {
				keep = false
			}

			if archFilter == "386" && c.Arch != Arch386 {
				keep = false
			}
		case FlavorNativeWindows:
			if osFilter != "windows" {
				keep = false
			}
		case FlavorNativeMacos:
			if osFilter != "darwin" {
				keep = false
			}
		}

		if keep {
			compatibleCandidates = append(compatibleCandidates, c)
		}
	}

	bestCandidates := compatibleCandidates

	if len(bestCandidates) == 1 {
		v.Candidates = bestCandidates
		return
	}

	// now keep all candidates of the lowest depth
	lowestDepth := 4096
	for _, c := range v.Candidates {
		if c.Depth < lowestDepth {
			lowestDepth = c.Depth
		}
	}

	bestCandidates = selectByFunc(compatibleCandidates, func(c *Candidate) bool {
		return c.Depth == lowestDepth
	})

	if len(bestCandidates) == 1 {
		v.Candidates = bestCandidates
		return
	}

	// love always wins, in the end
	{
		loveCandidates := selectByFlavor(bestCandidates, FlavorLove)

		if len(loveCandidates) == 1 {
			v.Candidates = loveCandidates
			return
		}
	}

	// on macOS, app bundles win
	if osFilter == "darwin" {
		appCandidates := selectByFlavor(bestCandidates, FlavorAppMacos)

		if len(appCandidates) > 0 {
			bestCandidates = appCandidates
		}
	}

	// on windows, scripts win
	if osFilter == "windows" {
		scriptCandidates := selectByFlavor(bestCandidates, FlavorScriptWindows)

		if len(scriptCandidates) == 1 {
			v.Candidates = scriptCandidates
			return
		}
	}

	// on linux, scripts win
	if osFilter == "linux" {
		scriptCandidates := selectByFlavor(bestCandidates, FlavorScript)

		if len(scriptCandidates) == 1 {
			v.Candidates = scriptCandidates
			return
		}
	}

	if osFilter == "linux" && archFilter == "amd64" {
		linuxCandidates := selectByFlavor(bestCandidates, FlavorNativeLinux)
		linux64Candidates := selectByArch(linuxCandidates, ArchAmd64)

		if len(linux64Candidates) > 0 {
			// on linux 64, 64-bit binaries win
			bestCandidates = linux64Candidates
		} else {
			// if no 64-bit binaries, jars win
			jarCandidates := selectByFlavor(bestCandidates, FlavorJar)
			if len(jarCandidates) > 0 {
				v.Candidates = jarCandidates
				return
			}
		}

		if len(bestCandidates) == 1 {
			v.Candidates = bestCandidates
			return
		}
	}

	// on windows, non-installers win
	if osFilter == "windows" {
		windowsCandidates := selectByFlavor(bestCandidates, FlavorNativeWindows)
		nonInstallerCandidates := selectByFunc(windowsCandidates, func(c *Candidate) bool {
			return !(c.WindowsInfo != nil && c.WindowsInfo.InstallerType != "")
		})

		if len(nonInstallerCandidates) > 0 {
			bestCandidates = nonInstallerCandidates
		}

		if len(bestCandidates) == 1 {
			v.Candidates = bestCandidates
			return
		}
	}

	// on windows, gui executables win
	if osFilter == "windows" {
		windowsCandidates := selectByFlavor(bestCandidates, FlavorNativeWindows)
		guiCandidates := selectByFunc(windowsCandidates, func(c *Candidate) bool {
			return c.WindowsInfo != nil && c.WindowsInfo.Gui
		})

		if len(guiCandidates) > 0 {
			bestCandidates = guiCandidates
		}

		if len(bestCandidates) == 1 {
			v.Candidates = bestCandidates
			return
		}
	}

	// everywhere, HTMLs lose if there's anything else good
	{
		htmlCandidates := selectByFlavor(bestCandidates, FlavorHTML)
		if len(htmlCandidates) > 0 && len(htmlCandidates) < len(bestCandidates) {
			bestCandidates = selectByFunc(bestCandidates, func(c *Candidate) bool {
				return c.Flavor != FlavorHTML
			})
		}
	}

	// everywhere, jars lose if there's anything else good
	{
		jarCandidates := selectByFlavor(bestCandidates, FlavorJar)
		if len(jarCandidates) > 0 && len(jarCandidates) < len(bestCandidates) {
			bestCandidates = selectByFunc(bestCandidates, func(c *Candidate) bool {
				return c.Flavor != FlavorJar
			})
		}
	}

	sort.Stable(&biggestFirst{bestCandidates})

	// score, filter & sort
	computeScore := func(candidate *Candidate) ScoredCandidate {
		var score int64 = 100
		for _, entry := range blacklist {
			if entry.pattern.MatchString(candidate.Path) {
				switch entry.penalty.kind {
				case PenaltyScore:
					score -= entry.penalty.delta
				case PenaltyExclude:
					score = 0
				}
			}
		}

		return ScoredCandidate{candidate, score}
	}

	var scoredCandidates []ScoredCandidate
	for _, candidate := range bestCandidates {
		scored := computeScore(candidate)
		if scored.score > 0 {
			scoredCandidates = append(scoredCandidates, scored)
		}
	}
	sort.Stable(&HighestScoreFirst{scoredCandidates})

	var finalCandidates []*Candidate
	for _, scored := range scoredCandidates {
		finalCandidates = append(finalCandidates, scored.candidate)
	}

	v.Candidates = finalCandidates
}
