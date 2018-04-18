package dash

import (
	"io"

	"github.com/itchio/spellbook"
	"github.com/itchio/wizardry/wizardry/wizutil"
)

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
