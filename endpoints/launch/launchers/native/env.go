package native

import (
	"fmt"
	"runtime"
	"strings"
)

// buildEnvBlock merges explicit launch environment values into the host
// environment without duplicate keys. This keeps os/exec's last-value lookup
// and sandbox runners' first-value lookup in agreement.
func buildEnvBlock(hostEnv []string, overrides map[string]string) []string {
	overriddenNames := make(map[string]struct{}, len(overrides))
	for name := range overrides {
		overriddenNames[normalizeEnvName(name)] = struct{}{}
	}

	result := make([]string, 0, len(hostEnv)+len(overrides))
	for _, entry := range hostEnv {
		name, _, found := strings.Cut(entry, "=")
		if found {
			if _, overridden := overriddenNames[normalizeEnvName(name)]; overridden {
				continue
			}
		}
		result = append(result, entry)
	}

	for name, value := range overrides {
		result = append(result, fmt.Sprintf("%s=%s", name, value))
	}
	return result
}

func normalizeEnvName(name string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(name)
	}
	return name
}
