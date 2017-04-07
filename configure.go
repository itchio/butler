package main

import (
	"io"
	"os"
	"path/filepath"
	"time"

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
	Flavor ExecFlavor
	Arch   string
}

func sniff(path string) (*SniffResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 8)
	_, err = io.ReadFull(f, buf)
	if err != nil {
		return nil, err
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
		return &SniffResult{
			Flavor: ExecElf,
		}, nil
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
		fullpath := filepath.Join(root, path)

		if !f.IsDir() {
			sRes, sErr := sniff(fullpath)
			if sErr != nil {
				comm.Logf("Could not sniff %s: %s", fullpath, sErr.Error())
			}

			if sRes != nil {
				comm.Logf("%s: %+v", fullpath, *sRes)
			}
		}

		return nil
	}

	filepath.Walk(root, walker)

	comm.Logf("Configured in %s", time.Since(startTime))
}
