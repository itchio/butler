package main

import (
	"debug/pe"
	"encoding/binary"

	"github.com/itchio/butler/comm"
)

func exeprops(path string) {
	f, err := pe.Open(path)
	must(err)

	comm.Logf("Machine: %x", f.Machine)
	comm.Logf("Number of symbols: %d", f.NumberOfSymbols)
	comm.Logf("Number of sections: %d", f.NumberOfSections)

	if f.Machine != pe.IMAGE_FILE_MACHINE_I386 {
		comm.Dief("only pe32 supported at this time")
	}

	oh, ok := (f.OptionalHeader).(*pe.OptionalHeader32)
	if !ok {
		comm.Dief("Could not cast optional header")
	}

	comm.Logf("Optional header number of rvas: %d", oh.NumberOfRvaAndSizes)
	for i := 0; i < int(oh.NumberOfRvaAndSizes); i++ {
		dd := oh.DataDirectory[i]
		comm.Logf("dd %d: %+v", i, dd)
	}

	il, err := f.ImportedLibraries()
	must(err)
	comm.Logf("Imported libraries: %v (%d items)", il, len(il))

	is, err := f.ImportedSymbols()
	must(err)
	comm.Logf("Imported symbols: %v (%d items)", is, len(is))

	for _, name := range []string{".text", ".bss", ".rdata", ".data", ".idata", ".reloc"} {
		section := f.Section(name)
		if section != nil {
			comm.Logf("Found %s section", name)
		}
	}

	rdata := f.Section(".rdata")
	if rdata != nil {
		comm.Logf("Found .rdata, trying to interpret")

		cd, err := rdata.Data()
		must(err)

		d := cd

		var ida []pe.ImportDirectory
		for len(d) > 0 {
			var dt pe.ImportDirectory
			dt.OriginalFirstThunk = binary.LittleEndian.Uint32(d[0:4])
			dt.Name = binary.LittleEndian.Uint32(d[12:16])
			dt.FirstThunk = binary.LittleEndian.Uint32(d[16:20])
			d = d[20:]
			if dt.OriginalFirstThunk == 0 {
				break
			}
			ida = append(ida, dt)
		}

		comm.Logf("Found %d elements", len(ida))
		for i, dt := range ida {
			start := int(dt.Name - rdata.VirtualAddress)
			if dll, success := getString(d, start); success {
				comm.Logf("Entry %d: dll %s", i, dll)
			}
		}
	}

	defer f.Close()
}

// getString extracts a string from symbol string table.
func getString(section []byte, start int) (string, bool) {
	if start < 0 || start >= len(section) {
		return "", false
	}

	for end := start; end < len(section); end++ {
		if section[end] == 0 {
			return string(section[start:end]), true
		}
	}
	return "", false
}
