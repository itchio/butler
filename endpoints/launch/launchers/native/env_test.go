package native

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildEnvBlockReplacesHostValues(t *testing.T) {
	hostEnv := []string{
		"LANG=en_US.UTF-8",
		"PATH=/usr/bin",
		"UNCHANGED=yes",
	}
	overrides := map[string]string{
		"LANG":  "ja_JP.UTF-8",
		"EMPTY": "",
	}

	result := buildEnvBlock(hostEnv, overrides)

	assert.Equal(t, []string{"ja_JP.UTF-8"}, envValues(result, "LANG"))
	assert.Equal(t, []string{"/usr/bin"}, envValues(result, "PATH"))
	assert.Equal(t, []string{"yes"}, envValues(result, "UNCHANGED"))
	assert.Equal(t, []string{""}, envValues(result, "EMPTY"))
}

func envValues(env []string, name string) []string {
	prefix := name + "="
	var values []string
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			values = append(values, strings.TrimPrefix(entry, prefix))
		}
	}
	return values
}
