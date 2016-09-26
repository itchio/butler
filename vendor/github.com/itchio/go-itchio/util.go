package itchio

import (
	"fmt"
	"strings"
)

// ParseSpec parses something of the form `user/page:channel` and returns
// `user/page` and `channel` separately
func ParseSpec(specIn string) (target string, channel string, err error) {
	spec := strings.ToLower(specIn)

	tokens := strings.Split(spec, ":")

	if len(tokens) == 1 {
		return "", "", fmt.Errorf("invalid spec: %s, missing channel (examples: %s:windows-32-beta, %s:linux-64)", spec, spec, spec)
	} else if len(tokens) != 2 {
		return "", "", fmt.Errorf("invalid spec: %s, expected something of the form user/page:channel", spec)
	}

	return tokens[0], tokens[1], nil
}
