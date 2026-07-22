package launch

import (
	"testing"

	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/ox"
)

func TestInteractionArchitecture(t *testing.T) {
	cases := map[string]itchio.SessionArchitecture{
		"amd64": itchio.SessionArchitectureAmd64,
		"arm64": itchio.SessionArchitectureArm64,
		"386":   itchio.SessionArchitecture386,
		"arm":   itchio.SessionArchitectureArm,
		"mips":  itchio.SessionArchitecture(""),
	}
	for arch, want := range cases {
		got := interactionArchitecture(ox.Runtime{Architecture: arch})
		if got != want {
			t.Errorf("arch %q: got %q, want %q", arch, got, want)
		}
	}
}
