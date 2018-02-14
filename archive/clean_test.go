package archive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// These tests are modified from Go's filepath tests

type PathTest struct {
	path, result string
}

var cleantests = []PathTest{
	// Already clean
	{"abc", "abc"},
	{"abc/def", "abc/def"},
	{"a/b/c", "a/b/c"},
	{".", "."},
	{"..", ".."},
	{"../..", "../.."},
	{"../../abc", "../../abc"},
	{"/abc", "/abc"},
	{"/", "/"},

	// Empty is current dir
	{"", "."},

	// Remove trailing slash
	{"abc/", "abc"},
	{"abc/def/", "abc/def"},
	{"a/b/c/", "a/b/c"},
	{"./", "."},
	{"../", ".."},
	{"../../", "../.."},
	{"/abc/", "/abc"},

	// Remove doubled slash
	{"abc//def//ghi", "abc/def/ghi"},
	{"//abc", "/abc"},
	{"///abc", "/abc"},
	{"//abc//", "/abc"},
	{"abc//", "abc"},

	// Remove . elements
	{"abc/./def", "abc/def"},
	{"/./abc/def", "/abc/def"},
	{"abc/.", "abc"},

	// Remove .. elements
	{"abc/def/ghi/../jkl", "abc/def/jkl"},
	{"abc/def/../ghi/../jkl", "abc/jkl"},
	{"abc/def/..", "abc"},
	{"abc/def/../..", "."},
	{"/abc/def/../..", "/"},
	{"abc/def/../../..", ".."},
	{"/abc/def/../../..", "/"},
	{"abc/def/../../../ghi/jkl/../../../mno", "../../mno"},
	{"/../abc", "/abc"},

	// Combinations
	{"abc/./../def", "def"},
	{"abc//./../def", "def"},
	{"abc/../../././../def", "../../def"},
	{`c:`, `c:.`},
	{`c:\`, `c:/`},
	{`c:\abc`, `c:/abc`},
	{`c:abc\..\..\.\.\..\def`, `c:../../def`},
	{`c:\abc\def\..\..`, `c:/`},
	{`c:\..\abc`, `c:/abc`},
	{`c:..\abc`, `c:../abc`},
	{`\`, `/`},
	{`/`, `/`},
	{`\\i\..\c$`, `/c$`},
	{`\\i\..\i\c$`, `/i/c$`},
	{`\\i\..\I\c$`, `/I/c$`},
	{`\\host\share\foo\..\bar`, `//host/share/bar`},
	{`//host/share/foo/../baz`, `//host/share/baz`},
	{`\\a\b\..\c`, `//a/b/c`},
	{`\\a\b`, `//a/b`},
}

func TestCleanFileName(t *testing.T) {
	for _, cas := range cleantests {
		assert.Equal(t, cas.result, CleanFileName(cas.path))
	}
}
