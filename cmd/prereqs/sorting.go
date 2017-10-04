package prereqs

import (
	"sort"
	"strings"

	"github.com/itchio/butler/redist"
)

type entriesByArch struct {
	entries []*NamedRedistEntry
}

var _ sort.Interface = (*entriesByArch)(nil)

func (ss *entriesByArch) Len() int {
	return len(ss.entries)
}

func (ss *entriesByArch) Less(i, j int) bool {
	return strings.Compare(ss.entries[i].entry.Arch, ss.entries[j].entry.Arch) == -1
}

func (ss *entriesByArch) Swap(i, j int) {
	ss.entries[i], ss.entries[j] = ss.entries[j], ss.entries[i]
}

type entriesByName struct {
	entries []*NamedRedistEntry
}

var _ sort.Interface = (*entriesByName)(nil)

func (ss *entriesByName) Len() int {
	return len(ss.entries)
}

func (ss *entriesByName) Less(i, j int) bool {
	return strings.Compare(ss.entries[i].name, ss.entries[j].name) == -1
}

func (ss *entriesByName) Swap(i, j int) {
	ss.entries[i], ss.entries[j] = ss.entries[j], ss.entries[i]
}

// NamedRedistEntry is used for sorting redists by names
type NamedRedistEntry struct {
	name  string
	entry *redist.RedistEntry
}
