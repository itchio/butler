package configurator

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fasterthanlime/spellbook"
	"github.com/go-errors/errors"
	"github.com/itchio/arkive/zip"
	"github.com/itchio/butler/comm"
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
	// FlavorScript denotes scripts starting with a shebang (#!)
	FlavorScript = "script"
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
	Path              string       `json:"path"`
	Mode              uint32       `json:"mode,omitempty"`
	Depth             int          `json:"depth"`
	Flavor            Flavor       `json:"flavor"`
	Arch              Arch         `json:"arch,omitempty"`
	Size              int64        `json:"size"`
	ImportedLibraries []string     `json:"importedLibraries,omitempty"`
	Spell             []string     `json:"spell,omitempty"`
	WindowsInfo       *WindowsInfo `json:"windows_info,omitempty"`
	LinuxInfo         *LinuxInfo   `json:"linux_info,omitempty"`
	MacosInfo         *MacosInfo   `json:"macos_info,omitempty"`
	LoveInfo          *LoveInfo    `json:"love_info,omitempty"`
	ScriptInfo        *ScriptInfo  `json:"script_info,omitempty"`
	JarInfo           *JarInfo     `json:"jar_info,omitempty"`
}

type WindowsInfo struct {
	InstallerType WindowsInstallerType `json:"installer_type,omitempty"`
	Gui           bool                 `json:"gui,omitempty"`
	DotNet        bool                 `json:"dotnet,omitempty"`
}

type WindowsInstallerType string

const (
	WindowsInstallerTypeMsi      WindowsInstallerType = "msi"
	WindowsInstallerTypeInno                          = "inno"
	WindowsInstallerTypeNullsoft                      = "nsis"
)

type MacosInfo struct {
}

type LinuxInfo struct {
	RequiredLibraries []string `json:"required_libraries,omitempty"`
}

type LoveInfo struct {
	Version string `json:"version,omitempty"`
}

type ScriptInfo struct {
	Interpreter string `json:"interpreter,omitempty"`
}

type JarInfo struct {
	MainClass string `json:"main_class,omitempty"`
}

type Verdict struct {
	BasePath   string       `json:"base_path"`
	TotalSize  int64        `json:"total_size"`
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
	spell := spellbook.Identify(&readerAtFromSeeker{r}, size, 0)

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
	} else if spellHas(spell, "\\b, Nullsoft Installer self-extracting archive") {
		result.WindowsInfo.InstallerType = WindowsInstallerTypeNullsoft
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
	spell := spellbook.Identify(&readerAtFromSeeker{r}, size, 0)

	if !spellHas(spell, "ELF") {
		// uh oh
		return nil, nil
	}

	if !spellHas(spell, "executable,") {
		// ignore libraries etc.
		return nil, nil
	}

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

func sniffLove(r io.ReadSeeker, size int64) (*Candidate, error) {
	res := &Candidate{
		Flavor:   FlavorLove,
		Path:     ".",
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
			defer rc.Close()

			if err != nil {
				// :(
				return nil, nil
			}

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

	spell := spellbook.Identify(ra, size, 0)

	if spellHas(spell, "compiled Java class data,") {
		// nevermind
		return nil, nil
	}

	return &Candidate{
		Flavor: FlavorNativeMacos,
		Spell:  spell,
	}, nil
}

func sniff(pool wsync.Pool, fileIndex int64, file *tlc.File) (*Candidate, error) {
	lowerPath := strings.ToLower(file.Path)

	r, err := pool.GetReadSeeker(fileIndex)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	size := pool.GetSize(fileIndex)

	switch lowerPath {
	case "index.html":
		return &Candidate{
			Flavor: FlavorHTML,
		}, nil
	case "conf.lua":
		return sniffLove(r, size)
	}

	buf := make([]byte, 8)
	n, _ := io.ReadFull(r, buf)
	if n < len(buf) {
		// too short to be an exec or unreadable
		return nil, nil
	}

	// if it ends in .exe, it's probably an .exe
	if strings.HasSuffix(strings.ToLower(file.Path), ".exe") {
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

func Configure(root string, showSpell bool, filterPaths tlc.FilterFunc) (*Verdict, error) {
	container, err := tlc.WalkAny(root, filterPaths)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	pool, err := pools.New(container, root)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	defer pool.Close()

	verdict := &Verdict{
		BasePath: root,
	}

	var candidates = make([]*Candidate, 0)

	for fileIndex, f := range container.Files {
		verdict.TotalSize += f.Size

		res, err := sniff(pool, int64(fileIndex), f)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		if res != nil {
			res.Size = f.Size
			if res.Path == "" {
				res.Path = f.Path
			}
			res.Mode = f.Mode

			res.Depth = len(strings.Split(f.Path, "/"))

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

func (v *Verdict) FilterPlatform(osFilter string, archFilter string) error {
	compatibleCandidates := make([]*Candidate, 0)

	// if we have any linux executables or scripts, make sure
	// we can execute them
	for _, c := range v.Candidates {
		switch c.Flavor {
		case FlavorNativeLinux, FlavorNativeMacos, FlavorScript:
			fullPath := filepath.Join(v.BasePath, c.Path)

			if c.Mode&0100 == 0 {
				comm.Logf("Fixing permissions for %s", c.Path)

				err := os.Chmod(fullPath, 0755)
				if err != nil {
					return err
				}
			}
		}

		c.Mode = 0
	}

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
		return nil
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
		return nil
	}

	// love always wins, in the end
	{
		loveCandidates := SelectByFlavor(bestCandidates, FlavorLove)

		if len(loveCandidates) == 1 {
			v.Candidates = loveCandidates
			return nil
		}
	}

	// on linux, scripts win
	if osFilter == "linux" {
		scriptCandidates := SelectByFlavor(bestCandidates, FlavorScript)

		if len(scriptCandidates) == 1 {
			v.Candidates = scriptCandidates
			return nil
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
				return nil
			}
		}

		if len(bestCandidates) == 1 {
			v.Candidates = bestCandidates
			return nil
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
			return nil
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
			return nil
		}
	}

	v.Candidates = bestCandidates
	return nil
}
