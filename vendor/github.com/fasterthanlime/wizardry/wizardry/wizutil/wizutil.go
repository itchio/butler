package wizutil

import (
	"regexp"
	"strings"
)

// IsWhitespace tests if a byte is either a space or a tab
func IsWhitespace(b byte) bool {
	return b == ' ' || b == '\t'
}

// IsNumber tests if a byte is in [0-9]
func IsNumber(b byte) bool {
	return '0' <= b && b <= '9'
}

// IsNumber tests if a byte is in [0-7]
func IsOctalNumber(b byte) bool {
	return '0' <= b && b <= '7'
}

// IsNumber tests if a byte is in [0-9A-Za-z]
func IsHexNumber(b byte) bool {
	return ('0' <= b && b <= '9') || ('a' <= b && b <= 'f') || ('A' <= b && b <= 'F')
}

// IsLowerLetter tests if a byte is in [a-z]
func IsLowerLetter(b byte) bool {
	return 'a' <= b && b <= 'z'
}

// IsUpperLetter tests if a byte is in [A-Z]
func IsUpperLetter(b byte) bool {
	return 'A' <= b && b <= 'Z'
}

// ToLower transliterates from [A-Z] to [a-z], other bytes are unchanged
func ToLower(b byte) byte {
	if IsUpperLetter(b) {
		return b + ('a' - 'A')
	}
	return b
}

// ToUpper transliterates from [a-z] to [A-Z], other bytes are unchanged
func ToUpper(b byte) byte {
	if IsLowerLetter(b) {
		return b - ('a' - 'A')
	}
	return b
}

// MergeStrings concatenates a set of strings return by Identify into
// a string that file(1) would print. For example, it handles \b.
func MergeStrings(outStrings []string) string {
	outString := strings.Join(outStrings, " ")

	re := regexp.MustCompile(`.\\b`)
	outString = re.ReplaceAllString(outString, "")
	outString = strings.TrimSpace(outString)

	return outString
}
