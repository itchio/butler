package version

import (
	"strconv"
	"strings"
)

// returns true if v1 >= v2
func GTOrEq(v1 string, v2 string) bool {
	v1toks := strings.Split(v1, ".")
	v2toks := strings.Split(v2, ".")

	for i, v1tok := range v1toks {
		if len(v2toks) <= i {
			// we have more tokens, we win
			return true
		}
		v2tok := v2toks[i]

		n1, _ := strconv.ParseInt(v1tok, 10, 64)
		n2, _ := strconv.ParseInt(v2tok, 10, 64)
		if n1 < n2 {
			return false
		}
	}
	return true
}
