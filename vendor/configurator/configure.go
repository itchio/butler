package configurator

import (
	"io"
	"time"

	"strings"

	"os"

	"encoding/json"

	"bufio"

	"regexp"

	"github.com/fasterthanlime/spellbook"
	"github.com/go-errors/errors"
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
	// FlavorScript denotes scripts starting with a script (#!)
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
	ArchI386  Arch = "i386"
	ArchAMD64      = "amd64"
)

// SniffResult indicates what's interesting about a file
type SniffResult struct {
	Path              string       `json:"path"`
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

type Verdict struct {
	Candidates []*SniffResult `json:"candidates"`
}

func spellHas(spell []string, token string) bool {
	for _, tok := range spell {
		if tok == token {
			return true
		}
	}
	return false
}

func sniffPE(r io.ReadSeeker, size int64) (*SniffResult, error) {
	spell := spellbook.Identify(&readerAtFromSeeker{r}, size, 0)

	if !spellHas(spell, "PE") {
		// uh oh
		return nil, nil
	}

	result := &SniffResult{
		Flavor:      FlavorNativeWindows,
		Spell:       spell,
		WindowsInfo: &WindowsInfo{},
	}

	if spellHas(spell, "\\b32 executable") {
		result.Arch = "386"
	} else if spellHas(spell, "\\b32+ executable") {
		result.Arch = "amd64"
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

func sniffELF(r io.ReadSeeker, size int64) (*SniffResult, error) {
	spell := spellbook.Identify(&readerAtFromSeeker{r}, size, 0)

	if !spellHas(spell, "ELF") {
		// uh oh
		return nil, nil
	}

	if !spellHas(spell, "executable,") {
		// ignore libraries etc.
		return nil, nil
	}

	result := &SniffResult{
		Flavor: FlavorNativeLinux,
		Spell:  spell,
	}

	if spellHas(spell, "32-bit") {
		result.Arch = "386"
	} else if spellHas(spell, "64-bit") {
		result.Arch = "amd64"
	}

	return result, nil
}

func sniffLove(r io.ReadSeeker, size int64) (*SniffResult, error) {
	res := &SniffResult{
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

func sniffScript(r io.ReadSeeker, size int64) (*SniffResult, error) {
	res := &SniffResult{
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

func sniff(pool wsync.Pool, fileIndex int64, file *tlc.File) (*SniffResult, error) {
	lowerPath := strings.ToLower(file.Path)

	r, err := pool.GetReadSeeker(fileIndex)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	size := pool.GetSize(fileIndex)

	switch lowerPath {
	case "index.html":
		return &SniffResult{
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
		return &SniffResult{
			Flavor: FlavorNativeMacos,
		}, nil
	}

	// Mach-O universal binaries start with 0xCAFEBABE
	// it's Apple's 'fat binary' stuff that contains multiple architectures
	if buf[0] == 0xCA && buf[1] == 0xFE && buf[2] == 0xBA && buf[3] == 0xBE {
		return &SniffResult{
			Flavor: FlavorNativeMacos,
		}, nil
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
		return &SniffResult{
			Flavor: FlavorNativeWindows,
			WindowsInfo: &WindowsInfo{
				InstallerType: WindowsInstallerTypeMsi,
			},
		}, nil
	}

	return nil, nil
}

func Configure(root string, showSpell bool, filterPaths tlc.FilterFunc) error {
	startTime := time.Now()

	comm.Opf("Walking container...")

	container, err := tlc.WalkAny(root, filterPaths)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	pool, err := pools.New(container, root)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	defer pool.Close()

	var candidates = make([]*SniffResult, 0)

	for fileIndex, f := range container.Files {
		res, err := sniff(pool, int64(fileIndex), f)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if res != nil {
			res.Size = f.Size
			if res.Path == "" {
				res.Path = f.Path
			}

			res.Depth = len(strings.Split(f.Path, "/"))

			candidates = append(candidates, res)
		}

		comm.Progress(float64(fileIndex) / float64(len(container.Files)))
	}

	comm.Statf("Configured in %s", time.Since(startTime))

	if !showSpell {
		for _, c := range candidates {
			c.Spell = nil
		}
	}

	verdict := &Verdict{
		Candidates: candidates,
	}

	marshalled, err := json.MarshalIndent(verdict, "", "  ")
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Logf("%s", string(marshalled))

	return nil
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
