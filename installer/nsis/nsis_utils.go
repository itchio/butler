package nsis

import (
	"fmt"
	"strings"
)

/*
 * Returns an array of arguments that will make an NSIS installer or uninstaller happy
 *
 * The docs say to "not wrap the argument in double quotes" but what they really mean is
 * just pass it as separate arguments (due to how f*cked argument parsing is)
 *
 * So this takes `/D=`, `C:\Itch Games\something` and returns
 * [`/D=C:\Itch`, `Games\something`]
 *
 * @param prefix something like `/D=` or `_?=` probably
 * @param path a path, may contain spaces, may not
 */
func getSeriouslyMisdesignedNsisPathArguments(prefix string, name string) []string {
	tokens := strings.Split(name, " ")
	tokens[0] = fmt.Sprintf("%s%s", prefix, tokens[0])
	return tokens
}
