package hades

import (
	"strings"
	"unicode"
)

func FromDBName(in string) string {
	var resTokens []string
	tokens := strings.Split(in, "_")
	for _, token := range tokens {
		resToken := strings.ToUpper(token[:1]) + token[1:]
		if resToken == "Id" {
			resToken = "ID"
		}
		resTokens = append(resTokens, resToken)
	}
	return strings.Join(resTokens, "")
}

func ToDBName(in string) string {
	runes := []rune(in)
	length := len(runes)

	var out []rune
	for i := 0; i < length; i++ {
		if i > 0 && unicode.IsUpper(runes[i]) && ((i+1 < length && unicode.IsLower(runes[i+1])) || unicode.IsLower(runes[i-1])) {
			out = append(out, '_')
		}
		out = append(out, unicode.ToLower(runes[i]))
	}

	return string(out)
}
