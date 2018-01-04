package configurator

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fasterthanlime/spellbook"
	"github.com/fasterthanlime/wizardry/wizardry/wizutil"
	"github.com/go-errors/errors"
	"github.com/itchio/arkive/zip"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/wharf/pools"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

// Flavor describes the flavor of an executable
type Flavor string

const (
	// FlavorNativeLinux denotes native linux executables
	FlavorNativeLinux Flavor = "linux"
	// ExecNativeMacos denotes native macOS executables
	FlavorNativeMacos = "macos"
	// FlavorPe denotes native windows executables
	FlavorNativeWindows = "windows"
	// FlavorAppMacos denotes a macOS app bundle
	FlavorAppMacos = "app-macos"
	// FlavorScript denotes scripts starting with a shebang (#!)
	FlavorScript = "script"
	// FlavorScriptWindows denotes windows scripts (.bat or .cmd)
	FlavorScriptWindows = "windows-script"
	// FlavorJar denotes a .jar archive with a Main-Class
	FlavorJar = "jar"
	// FlavorHTML denotes an index html file
	FlavorHTML = "html"
	// FlavorLove denotes a love package
	FlavorLove = "love"
)

type Arch string

const (
	Arch386   Arch = "386"
	ArchAmd64      = "amd64"
)

// Candidate indicates what's interesting about a file
type Candidate struct {
	Path        string       `json:"path"`
	Mode        uint32       `json:"mode,omitempty"`
	Depth       int          `json:"depth"`
	Flavor      Flavor       `json:"flavor"`
	Arch        Arch         `json:"arch,omitempty"`
	Size        int64        `json:"size"`
	Spell       []string     `json:"spell,omitempty"`
	WindowsInfo *WindowsInfo `json:"windowsInfo,omitempty"`
	LinuxInfo   *LinuxInfo   `json:"linuxInfo,omitempty"`
	MacosInfo   *MacosInfo   `json:"macosInfo,omitempty"`
	LoveInfo    *LoveInfo    `json:"loveInfo,omitempty"`
	ScriptInfo  *ScriptInfo  `json:"scriptInfo,omitempty"`
	JarInfo     *JarInfo     `json:"jarInfo,omitempty"`
}

func (c *Candidate) String() string {
	marshalled, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return ""
	}

	return string(marshalled)
}

type WindowsInfo struct {
	InstallerType WindowsInstallerType `json:"installerType,omitempty"`
	Uninstaller   bool                 `json:"uninstaller,omitempty"`
	Gui           bool                 `json:"gui,omitempty"`
	DotNet        bool                 `json:"dotNet,omitempty"`
}

type WindowsInstallerType string

const (
	WindowsInstallerTypeMsi      WindowsInstallerType = "msi"
	WindowsInstallerTypeInno                          = "inno"
	WindowsInstallerTypeNullsoft                      = "nsis"
	// self-extracting installer that unarchiver knows how to extract
	WindowsInstallerTypeArchive = "archive"
)

type MacosInfo struct {
}

type LinuxInfo struct {
}

type LoveInfo struct {
	Version string `json:"version,omitempty"`
}

type ScriptInfo struct {
	Interpreter string `json:"interpreter,omitempty"`
}

type JarInfo struct {
	MainClass string `json:"mainClass,omitempty"`
}

type Verdict struct {
	BasePath   string       `json:"basePath"`
	TotalSize  int64        `json:"totalSize"`
	Candidates []*Candidate `json:"candidates"`
}

func (v *Verdict) String() string {
	marshalled, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}

	return string(marshalled)
}

func spellHas(spell []string, token string) bool {
	for _, tok := range spell {
		if tok == token {
			return true
		}
	}
	return false
}

func sniffPE(r io.ReadSeeker, size int64) (*Candidate, error) {
	sr := wizutil.NewSliceReader(&readerAtFromSeeker{r}, 0, size)
	spell := spellbook.Identify(sr, 0)

	if !spellHas(spell, "PE") {
		// uh oh
		return nil, nil
	}

	result := &Candidate{
		Flavor:      FlavorNativeWindows,
		Spell:       spell,
		WindowsInfo: &WindowsInfo{},
	}

	if spellHas(spell, "\\b32 executable") {
		result.Arch = Arch386
	} else if spellHas(spell, "\\b32+ executable") {
		result.Arch = ArchAmd64
	}

	if spellHas(spell, "\\b, InnoSetup installer") {
		result.WindowsInfo.InstallerType = WindowsInstallerTypeInno
	} else if spellHas(spell, "\\b, InnoSetup uninstaller") {
		result.WindowsInfo.InstallerType = WindowsInstallerTypeInno
		result.WindowsInfo.Uninstaller = true
	} else if spellHas(spell, "\\b, Nullsoft Installer self-extracting archive") {
		result.WindowsInfo.InstallerType = WindowsInstallerTypeNullsoft
	} else if spellHas(spell, "\\b, InstallShield self-extracting archive") {
		result.WindowsInfo.InstallerType = WindowsInstallerTypeArchive
	}

	if spellHas(spell, "(GUI)") {
		result.WindowsInfo.Gui = true
	}

	if spellHas(spell, "Mono/.Net assembly") {
		result.WindowsInfo.DotNet = true
	}

	return result, nil
}

func sniffELF(r io.ReadSeeker, size int64) (*Candidate, error) {
	sr := wizutil.NewSliceReader(&readerAtFromSeeker{r}, 0, size)
	spell := spellbook.Identify(sr, 0)

	if !spellHas(spell, "ELF") {
		// uh oh
		return nil, nil
	}

	// some objects are marked as 'executable', others are marked
	// as 'shared objects', but it doesn't matter since executables
	// can be marked as shared objects as well (node-webkit) for example.

	result := &Candidate{
		Flavor: FlavorNativeLinux,
		Spell:  spell,
	}

	if spellHas(spell, "32-bit") {
		result.Arch = Arch386
	} else if spellHas(spell, "64-bit") {
		result.Arch = ArchAmd64
	}

	return result, nil
}

func sniffLove(r io.ReadSeeker, size int64, path string) (*Candidate, error) {
	res := &Candidate{
		Flavor:   FlavorLove,
		Path:     path,
		LoveInfo: &LoveInfo{},
	}

	s := bufio.NewScanner(r)

	re := regexp.MustCompile(`t\.version\s*=\s*"([^"]+)"`)

	for s.Scan() {
		line := s.Bytes()
		matches := re.FindSubmatch(line)
		if len(matches) == 2 {
			res.LoveInfo.Version = string(matches[1])
			break
		}
	}

	return res, nil
}

func sniffScript(r io.ReadSeeker, size int64) (*Candidate, error) {
	res := &Candidate{
		Flavor:     FlavorScript,
		ScriptInfo: &ScriptInfo{},
	}

	_, err := r.Seek(0, os.SEEK_SET)
	if err != nil {
		return nil, err
	}

	s := bufio.NewScanner(r)

	if s.Scan() {
		line := s.Text()
		if len(line) > 2 {
			// skip over the shebang
			res.ScriptInfo.Interpreter = strings.TrimSpace(line[2:])
		}
	}

	return res, nil
}

func sniffZip(r io.ReadSeeker, size int64) (*Candidate, error) {
	ra := &readerAtFromSeeker{r}

	zr, err := zip.NewReader(ra, size)
	if err != nil {
		// not a zip, probably
		return nil, nil
	}

	for _, f := range zr.File {
		path := filepath.ToSlash(filepath.Clean(filepath.ToSlash(f.Name)))
		if path == "META-INF/MANIFEST.MF" {
			rc, err := f.Open()
			if err != nil {
				// :(
				return nil, nil
			}
			defer rc.Close()

			s := bufio.NewScanner(rc)

			for s.Scan() {
				tokens := strings.SplitN(s.Text(), ":", 2)
				if len(tokens) > 0 && tokens[0] == "Main-Class" {
					mainClass := strings.TrimSpace(tokens[1])
					res := &Candidate{
						Flavor: FlavorJar,
						JarInfo: &JarInfo{
							MainClass: mainClass,
						},
					}
					return res, nil
				}
			}

			// we found the manifest, even if we couldn't read it
			// or it didn't have a main class
			break
		}
	}

	return nil, nil
}

func sniffFatMach(r io.ReadSeeker, size int64) (*Candidate, error) {
	ra := &readerAtFromSeeker{r}

	sr := wizutil.NewSliceReader(ra, 0, size)
	spell := spellbook.Identify(sr, 0)

	if spellHas(spell, "compiled Java class data,") {
		// nevermind
		return nil, nil
	}

	return &Candidate{
		Flavor: FlavorNativeMacos,
		Spell:  spell,
	}, nil
}

func sniffPoolEntry(pool wsync.Pool, fileIndex int64, file *tlc.File) (*Candidate, error) {
	r, err := pool.GetReadSeeker(fileIndex)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	size := pool.GetSize(fileIndex)

	return Sniff(r, file.Path, size)
}

func Sniff(r io.ReadSeeker, path string, size int64) (*Candidate, error) {
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

	// if it ends in .exe, it's probably an .exe
	if strings.HasSuffix(lowerPath, ".exe") {
		subRes, subErr := sniffPE(r, size)
		if subErr != nil {
			return nil, errors.Wrap(subErr, 0)
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

func pathToDepth(path string) int {
	return len(strings.Split(path, "/"))
}

func Configure(root string, showSpell bool) (*Verdict, error) {
	verdict := &Verdict{
		BasePath: root,
	}

	var pool wsync.Pool

	container, err := tlc.WalkAny(root, &tlc.WalkOpts{Filter: filtering.FilterPaths})
	if err != nil {
		comm.Logf("Could not walk %s: %s", root, err.Error())
		return verdict, nil
	}

	if pool == nil {
		pool, err = pools.New(container, root)
		if err != nil {
			return nil, errors.Wrap(err, 0)
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
				comm.Logf("Found app bundle without an Info.plist: %s", d.Path)
				continue
			}

			res := &Candidate{
				Flavor: FlavorAppMacos,
				Size:   0,
				Path:   d.Path,
				Mode:   d.Mode,
			}
			res.Depth = pathToDepth(res.Path)
			candidates = append(candidates, res)
		}
	}

	for fileIndex, f := range container.Files {
		verdict.TotalSize += f.Size

		res, err := sniffPoolEntry(pool, int64(fileIndex), f)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		if res != nil {
			res.Size = f.Size
			if res.Path == "" {
				res.Path = f.Path
			}
			res.Mode = f.Mode

			res.Depth = pathToDepth(res.Path)

			candidates = append(candidates, res)
		}
	}

	verdict.Candidates = candidates

	if !showSpell {
		for _, c := range candidates {
			c.Spell = nil
		}
	}

	return verdict, nil
}

// Adapt an io.ReadSeeker into an io.ReaderAt in the dumbest possible fashion

type readerAtFromSeeker struct {
	rs io.ReadSeeker
}

var _ io.ReaderAt = (*readerAtFromSeeker)(nil)

func (r *readerAtFromSeeker) ReadAt(b []byte, off int64) (int, error) {
	_, err := r.rs.Seek(off, os.SEEK_SET)
	if err != nil {
		return 0, err
	}

	return r.rs.Read(b)
}

func SelectByFlavor(candidates []*Candidate, f Flavor) []*Candidate {
	res := make([]*Candidate, 0)
	for _, c := range candidates {
		if c.Flavor == f {
			res = append(res, c)
		}
	}
	return res
}

func SelectByArch(candidates []*Candidate, a Arch) []*Candidate {
	res := make([]*Candidate, 0)
	for _, c := range candidates {
		if c.Arch == a {
			res = append(res, c)
		}
	}
	return res
}

type CandidateFilter func(candidate *Candidate) bool

func SelectByFunc(candidates []*Candidate, f CandidateFilter) []*Candidate {
	res := make([]*Candidate, 0)
	for _, c := range candidates {
		if f(c) {
			res = append(res, c)
		}
	}
	return res
}

func (v *Verdict) FixPermissions(dryrun bool) ([]string, error) {
	var fixed []string

	// if we have any linux executables or scripts, make sure
	// we can execute them
	for _, c := range v.Candidates {
		switch c.Flavor {
		case FlavorNativeLinux, FlavorNativeMacos, FlavorScript:
			fullPath := filepath.Join(v.BasePath, c.Path)

			if c.Mode&0100 == 0 {
				comm.Logf("Fixing permissions for %s", c.Path)

				fixed = append(fixed, c.Path)
				if !dryrun {
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

type BiggestFirst struct {
	candidates []*Candidate
}

var _ sort.Interface = (*BiggestFirst)(nil)

func (bf *BiggestFirst) Len() int {
	return len(bf.candidates)
}

func (bf *BiggestFirst) Less(i, j int) bool {
	return bf.candidates[i].Size > bf.candidates[j].Size
}

func (bf *BiggestFirst) Swap(i, j int) {
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
}

type ScoredCandidate struct {
	candidate *Candidate
	score     int64
}

func (v *Verdict) FilterPlatform(osFilter string, archFilter string) {
	compatibleCandidates := make([]*Candidate, 0)

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

	bestCandidates = SelectByFunc(compatibleCandidates, func(c *Candidate) bool {
		return c.Depth == lowestDepth
	})

	if len(bestCandidates) == 1 {
		v.Candidates = bestCandidates
		return
	}

	// love always wins, in the end
	{
		loveCandidates := SelectByFlavor(bestCandidates, FlavorLove)

		if len(loveCandidates) == 1 {
			v.Candidates = loveCandidates
			return
		}
	}

	// on macOS, app bundles win
	if osFilter == "darwin" {
		appCandidates := SelectByFlavor(bestCandidates, FlavorAppMacos)

		if len(appCandidates) > 0 {
			bestCandidates = appCandidates
		}
	}

	// on windows, scripts win
	if osFilter == "windows" {
		scriptCandidates := SelectByFlavor(bestCandidates, FlavorScriptWindows)

		if len(scriptCandidates) == 1 {
			v.Candidates = scriptCandidates
			return
		}
	}

	// on linux, scripts win
	if osFilter == "linux" {
		scriptCandidates := SelectByFlavor(bestCandidates, FlavorScript)

		if len(scriptCandidates) == 1 {
			v.Candidates = scriptCandidates
			return
		}
	}

	if osFilter == "linux" && archFilter == "amd64" {
		linuxCandidates := SelectByFlavor(bestCandidates, FlavorNativeLinux)
		linux64Candidates := SelectByArch(linuxCandidates, ArchAmd64)

		if len(linux64Candidates) > 0 {
			// on linux 64, 64-bit binaries win
			bestCandidates = linux64Candidates
		} else {
			// if no 64-bit binaries, jars win
			jarCandidates := SelectByFlavor(bestCandidates, FlavorJar)
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
		windowsCandidates := SelectByFlavor(bestCandidates, FlavorNativeWindows)
		nonInstallerCandidates := SelectByFunc(windowsCandidates, func(c *Candidate) bool {
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
		windowsCandidates := SelectByFlavor(bestCandidates, FlavorNativeWindows)
		guiCandidates := SelectByFunc(windowsCandidates, func(c *Candidate) bool {
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
		htmlCandidates := SelectByFlavor(bestCandidates, FlavorHTML)
		if len(htmlCandidates) > 0 && len(htmlCandidates) < len(bestCandidates) {
			bestCandidates = SelectByFunc(bestCandidates, func(c *Candidate) bool {
				return c.Flavor != FlavorHTML
			})
		}
	}

	// everywhere, jars lose if there's anything else good
	{
		jarCandidates := SelectByFlavor(bestCandidates, FlavorJar)
		if len(jarCandidates) > 0 && len(jarCandidates) < len(bestCandidates) {
			bestCandidates = SelectByFunc(bestCandidates, func(c *Candidate) bool {
				return c.Flavor != FlavorJar
			})
		}
	}

	// sort by biggest first
	sort.Stable(&BiggestFirst{bestCandidates})

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
