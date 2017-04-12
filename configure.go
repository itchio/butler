package main

import (
	"debug/elf"
	"debug/pe"
	"io"
	"os"
	"path/filepath"
	"time"

	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
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
)

// SniffResult indicates what's interesting about a file
type SniffResult struct {
	Flavor            ExecFlavor
	Arch              string
	Size              int64
	ImportedLibraries []string
}

func sniffPE(path string, f *os.File) (*SniffResult, error) {
	pf, err := pe.NewFile(f)
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

	if oh32, ok := pf.OptionalHeader.(*pe.OptionalHeader32); ok {
		comm.Logf("%s: found optional header 32", path)
		comm.Logf("%s: loader flags = %d", path, oh32.LoaderFlags)
		comm.Logf("%s: subsystem = %d", path, oh32.Subsystem)
	} else {
		comm.Logf("%s: no optional header 32")
		if oh64, ok := pf.OptionalHeader.(*pe.OptionalHeader64); ok {
			comm.Logf("%s: found optional header 64", path)
			comm.Logf("%s: loader flags = %d", path, oh64.LoaderFlags)
			comm.Logf("%s: subsystem = %d", path, oh64.Subsystem)
		}
	}

	libs, err := pf.ImportedLibraries()
	if err != nil {
		comm.Logf("Could not get imported libraries for %s", path)
	} else {
		result.ImportedLibraries = libs
	}

	return result, nil
}

func sniffELF(path string, f *os.File) (*SniffResult, error) {
	base := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(base, ".so") || strings.Contains(base, ".so.") {
		// shared library, not an executable, ignore
		return nil, nil
	}

	ef, err := elf.Open(path)
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
		comm.Logf("Could not get imported libraries for %s", path)
	} else {
		result.ImportedLibraries = libs
	}

	return result, nil
}

func sniff(path string) (*SniffResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	defer f.Close()

	buf := make([]byte, 8)
	_, err = io.ReadFull(f, buf)
	if err != nil {
		if errors.Is(err, io.EOF) {
			// too short to be an exec
			return nil, nil
		}
		return nil, errors.Wrap(err, 0)
	}

	// if it ends in .exe, it's probably an .exe
	if strings.HasSuffix(strings.ToLower(path), ".exe") {
		subRes, subErr := sniffPE(path, f)
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
		return sniffELF(path, f)
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
	startTime := time.Now()

	walker := func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !f.IsDir() {
			sRes, sErr := sniff(path)
			if sErr != nil {
				switch sErr := sErr.(type) {
				case *errors.Error:
					comm.Logf("Could not sniff %s: %s", path, sErr.ErrorStack())
				default:
					comm.Logf("Could not sniff %s: %s", path, sErr.Error())
				}
			}

			if sRes != nil {
				sRes.Size = f.Size()
				if sRes.Arch == "" {
					sRes.Arch = "any"
				}
				comm.Logf("%s %v", path, *sRes)
			}
		}

		return nil
	}

	filepath.Walk(root, walker)

	comm.Logf("Configured in %s", time.Since(startTime))
}
