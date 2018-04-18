package dash

import (
	"io"

	"github.com/itchio/spellbook"
	"github.com/itchio/wizardry/wizardry/wizutil"
)

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
