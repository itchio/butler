package butlerd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyCommandTemplateEmpty(t *testing.T) {
	executable, args, env, environmentOverrides, err := ApplyCommandTemplate("", "/games/my game", []string{"--manifest"}, map[string]string{"ITCHIO_APP": "1"})
	require.NoError(t, err)
	assert.Equal(t, "/games/my game", executable)
	assert.Equal(t, []string{"--manifest"}, args)
	assert.Equal(t, map[string]string{"ITCHIO_APP": "1"}, env)
	assert.Empty(t, environmentOverrides)
}

func TestApplyCommandTemplateAppendsArgumentsWithoutPlaceholder(t *testing.T) {
	executable, args, _, _, err := ApplyCommandTemplate(`--fullscreen --profile "Player One"`, "game", []string{"--manifest"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "game", executable)
	assert.Equal(t, []string{"--manifest", "--fullscreen", "--profile", "Player One"}, args)
}

func TestApplyCommandTemplateWrapsResolvedCommand(t *testing.T) {
	executable, args, env, environmentOverrides, err := ApplyCommandTemplate(
		`MESA_GL_VERSION_OVERRIDE=4.5 PROFILE="Player One" gamescope -f -- %command% --fullscreen`,
		"/usr/bin/java",
		[]string{"-jar", "/games/My Game/game.jar", "--manifest"},
		map[string]string{"PROFILE": "old", "ITCHIO_APP": "1"},
	)
	require.NoError(t, err)
	assert.Equal(t, "gamescope", executable)
	assert.Equal(t, []string{
		"-f", "--", "/usr/bin/java", "-jar", "/games/My Game/game.jar", "--manifest", "--fullscreen",
	}, args)
	assert.Equal(t, map[string]string{
		"ITCHIO_APP":               "1",
		"MESA_GL_VERSION_OVERRIDE": "4.5",
		"PROFILE":                  "Player One",
	}, env)
	assert.Equal(t, []string{"MESA_GL_VERSION_OVERRIDE", "PROFILE"}, environmentOverrides)
}

func TestApplyCommandTemplateOnlyEnvironment(t *testing.T) {
	executable, args, env, environmentOverrides, err := ApplyCommandTemplate("DEBUG=1", "game", []string{"--manifest"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "game", executable)
	assert.Equal(t, []string{"--manifest"}, args)
	assert.Equal(t, map[string]string{"DEBUG": "1"}, env)
	assert.Equal(t, []string{"DEBUG"}, environmentOverrides)
}

func TestApplyCommandTemplateRejectsMultiplePlaceholders(t *testing.T) {
	_, _, _, _, err := ApplyCommandTemplate("wrapper %command% %command%", "game", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most one")
}

func TestApplyCommandTemplateRejectsMalformedQuotes(t *testing.T) {
	_, _, _, _, err := ApplyCommandTemplate(`%command% "unterminated`, "game", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid command template")
}

func TestApplyCommandTemplateRequiresStandalonePlaceholder(t *testing.T) {
	executable, args, _, _, err := ApplyCommandTemplate("wrapper prefix-%command%", "game", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "game", executable)
	assert.Equal(t, []string{"wrapper", "prefix-%command%"}, args)
}

func TestApplyCommandTemplateRejectsEmptyWrapperExecutable(t *testing.T) {
	_, _, _, _, err := ApplyCommandTemplate(`'' %command%`, "game", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wrapper executable cannot be empty")
}
