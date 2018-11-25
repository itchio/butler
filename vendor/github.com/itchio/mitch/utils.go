package mitch

import (
	"fmt"
	"regexp"
	"strings"
)

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("fatal error: %+v", err))
	}
}

func (s *Store) serial() int64 {
	s.idSeed += 100
	return s.idSeed
}

var (
	invalidUsernameChars = regexp.MustCompile("^[A-Za-z0-9_]")
)

func (s *Store) slugify(input string) string {
	var res = input
	res = strings.ToLower(res)
	res = invalidUsernameChars.ReplaceAllString(res, "_")
	return res
}
