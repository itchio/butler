package butlerd

import (
	"fmt"
	"regexp"
	"strings"

	shellquote "github.com/kballard/go-shellquote"
)

const commandPlaceholder = "%command%"

var environmentVariableName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidateCommandTemplate checks that a command template is syntactically
// valid. Templates may contain zero or one standalone %command% token.
func ValidateCommandTemplate(commandTemplate string) error {
	_, err := parseCommandTemplate(commandTemplate)
	return err
}

// ApplyCommandTemplate applies a user-provided template to an already-resolved
// command. A standalone %command% token is replaced by the executable and all
// its arguments. When the placeholder is omitted, template tokens are appended
// as arguments. Leading NAME=value tokens become environment overrides.
func ApplyCommandTemplate(commandTemplate string, executable string, args []string, env map[string]string) (string, []string, map[string]string, []string, error) {
	template, err := parseCommandTemplate(commandTemplate)
	if err != nil {
		return "", nil, nil, nil, err
	}

	resultEnv := make(map[string]string, len(env)+len(template.env))
	for name, value := range env {
		resultEnv[name] = value
	}
	environmentOverrides := make([]string, 0, len(template.env))
	for _, assignment := range template.env {
		name := assignment.name
		value := assignment.value
		resultEnv[name] = value
		environmentOverrides = append(environmentOverrides, name)
	}

	if template.placeholderIndex < 0 {
		resultArgs := append([]string{}, args...)
		resultArgs = append(resultArgs, template.tokens...)
		return executable, resultArgs, resultEnv, environmentOverrides, nil
	}

	result := make([]string, 0, len(template.tokens)+len(args))
	result = append(result, template.tokens[:template.placeholderIndex]...)
	result = append(result, executable)
	result = append(result, args...)
	result = append(result, template.tokens[template.placeholderIndex+1:]...)
	if len(result) == 0 || result[0] == "" {
		return "", nil, nil, nil, fmt.Errorf("command template produced an empty command")
	}

	return result[0], result[1:], resultEnv, environmentOverrides, nil
}

type parsedCommandTemplate struct {
	tokens           []string
	env              []environmentAssignment
	placeholderIndex int
}

type environmentAssignment struct {
	name  string
	value string
}

func parseCommandTemplate(commandTemplate string) (*parsedCommandTemplate, error) {
	tokens, err := shellquote.Split(commandTemplate)
	if err != nil {
		return nil, fmt.Errorf("invalid command template: %w", err)
	}

	result := &parsedCommandTemplate{
		placeholderIndex: -1,
	}

	for len(tokens) > 0 {
		name, value, ok := parseEnvironmentAssignment(tokens[0])
		if !ok {
			break
		}
		result.env = append(result.env, environmentAssignment{name: name, value: value})
		tokens = tokens[1:]
	}
	result.tokens = tokens

	for index, token := range tokens {
		if token != commandPlaceholder {
			continue
		}
		if result.placeholderIndex >= 0 {
			return nil, fmt.Errorf("command template may contain at most one standalone %s token", commandPlaceholder)
		}
		result.placeholderIndex = index
	}
	if result.placeholderIndex > 0 && result.tokens[0] == "" {
		return nil, fmt.Errorf("command template wrapper executable cannot be empty")
	}

	return result, nil
}

func parseEnvironmentAssignment(token string) (string, string, bool) {
	name, value, found := strings.Cut(token, "=")
	if !found || !environmentVariableName.MatchString(name) {
		return "", "", false
	}
	return name, value, true
}
