package itchio

import (
	"fmt"
	"strings"
)

type Spec struct {
	Target  string
	Channel string
}

// ParseSpec parses something of the form `user/page:channel` and returns
// `user/page` and `channel` separately
func ParseSpec(specIn string) (*Spec, error) {
	specStr := strings.ToLower(specIn)
	tokens := strings.Split(specStr, ":")

	spec := &Spec{}

	switch len(tokens) {
	case 1:
		// no channel
		spec.Target = tokens[0]
	case 2:
		spec.Target = tokens[0]
		spec.Channel = tokens[1]
	default:
		return nil, fmt.Errorf("invalid spec: %s, expected something of the form user/page:channel", spec)
	}

	return spec, nil
}

func (spec *Spec) EnsureChannel() error {
	if spec.Channel == "" {
		return fmt.Errorf("invalid spec: %s, missing channel (examples: %s:windows-32-beta, %s:linux-64)", spec, spec, spec)
	}

	return nil
}
