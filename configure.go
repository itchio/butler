package main

import (
	"debug/elf"
	"debug/pe"
	"io"
	"time"

	"strings"

	"os"

	"encoding/json"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/pools"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

// ExecFlavor describes the flavor of an executable
type ExecFlavor string

const (
	// ExecElf denotes native linux executables
	ExecElf ExecFlavor = "elf"
	// ExecMachO denotes native macOS executables
	ExecMachO = "mach-o"
	// ExecPe denotes native windows executables
	ExecPe = "pe"
	// ExecShebang denotes scripts starting with a shebang (#!)
	ExecShebang = "shebang"
	// ExecJar denotes a .jar archive with a Main-Class
	ExecJar = "jar"
	// ExecMsi denotes a microsoft installer package
	ExecMsi = "msi"
	// ExecHTML denotes an index html file
	ExecHTML = "html"
)

// SniffResult indicates what's interesting about a file
type SniffResult struct {
	Path              string     `json:"path"`
	Flavor            ExecFlavor `json:"flavor"`
	Arch              string     `json:"arch,omitempty"`
	Size              int64      `json:"size"`
	ImportedLibraries []string   `json:"importedLibraries,omitempty"`
}

type Verdict struct {
	Candidates []*SniffResult `json:"candidates"`
}

func sniffPE(r io.ReaderAt) (*SniffResult, error) {
	pf, err := pe.NewFile(r)
	if err != nil {
		if errors.Is(err, io.EOF) {
			// yeah not an exe
			return nil, nil
		}
		// something else went wrong
		return nil, errors.Wrap(err, 0)
	}

	result := &SniffResult{
		Flavor: ExecPe,
	}

	switch pf.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		result.Arch = "386"
	case pe.IMAGE_FILE_MACHINE_AMD64:
		result.Arch = "amd64"
	}

	return result, nil
}

func sniffELF(r io.ReaderAt) (*SniffResult, error) {
	// base := strings.ToLower(filepath.Base(path))
	// if strings.HasSuffix(base, ".so") || strings.Contains(base, ".so.") {
	// 	// shared library, not an executable, ignore
	// 	return nil, nil
	// }

	ef, err := elf.NewFile(r)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	result := &SniffResult{
		Flavor: ExecElf,
	}

	switch ef.Machine {
	case elf.EM_386:
		result.Arch = "386"
	case elf.EM_X86_64:
		result.Arch = "amd64"
	}

	libs, err := ef.ImportedLibraries()
	if err != nil {
		comm.Logf("Could not get imported libraries for ELF")
	} else {
		result.ImportedLibraries = libs
	}

	return result, nil
}

func sniff(pool wsync.Pool, fileIndex int64, file *tlc.File) (*SniffResult, error) {
	if strings.ToLower(file.Path) == "index.html" {
		return &SniffResult{
			Flavor: ExecHTML,
		}, nil
	}

	r, err := pool.GetReadSeeker(fileIndex)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	buf := make([]byte, 8)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		if errors.Is(err, io.EOF) {
			// too short to be an exec
			return nil, nil
		}
		return nil, errors.Wrap(err, 0)
	}

	ra := &readerAtFromSeeker{r}

	// if it ends in .exe, it's probably an .exe
	if strings.HasSuffix(strings.ToLower(file.Path), ".exe") {
		subRes, subErr := sniffPE(ra)
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
			Flavor: ExecMachO,
		}, nil
	}

	// Mach-O universal binaries start with 0xCAFEBABE
	// it's Apple's 'fat binary' stuff that contains multiple architectures
	if buf[0] == 0xCA && buf[1] == 0xFE && buf[2] == 0xBA && buf[3] == 0xBE {
		return &SniffResult{
			Flavor: ExecMachO,
		}, nil
	}

	// ELF executables start with 0x7F454C46
	// (e.g. 0x7F + 'ELF' in ASCII)
	if buf[0] == 0x7F && buf[1] == 0x45 && buf[2] == 0x4C && buf[3] == 0x46 {
		return sniffELF(ra)
	}

	// Shell scripts start with a shebang (#!)
	// https://en.wikipedia.org/wiki/Shebang_(Unix)
	if buf[0] == 0x23 && buf[1] == 0x21 {
		return &SniffResult{
			Flavor: ExecShebang,
		}, nil
	}

	// MSI (Microsoft Installer Packages) have a well-defined magic number.
	if buf[0] == 0xD0 && buf[1] == 0xCF &&
		buf[2] == 0x11 && buf[3] == 0xE0 &&
		buf[4] == 0xA1 && buf[5] == 0xB1 &&
		buf[6] == 0x1A && buf[7] == 0xE1 {
		return &SniffResult{
			Flavor: ExecMsi,
		}, nil
	}

	return nil, nil
}

func configure(root string) {
	must(doConfigure(root))
}

func doConfigure(root string) error {
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

	var candidates []*SniffResult

	comm.StartProgress()

	for fileIndex, f := range container.Files {
		res, err := sniff(pool, int64(fileIndex), f)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if res != nil {
			res.Size = f.Size
			if res.Arch == "" {
				res.Arch = "any"
			}
			res.Path = f.Path

			candidates = append(candidates, res)
		}

		comm.Progress(float64(fileIndex) / float64(len(container.Files)))
	}

	comm.EndProgress()

	comm.Statf("Configured in %s", time.Since(startTime))

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
