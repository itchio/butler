package bfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_DirTreePrivate(t *testing.T) {
	dt := NewDirTree("/usr")
	assert.False(t, dt.hasPath("lib/a"))

	dt.commitPath("lib/a/b/foo/")
	assert.True(t, dt.hasPath("lib"))
	assert.True(t, dt.hasPath("lib/a"))
	assert.True(t, dt.hasPath("lib/a/b"))
	assert.True(t, dt.hasPath("lib/a/b/foo"))
	assert.False(t, dt.hasPath("lib/a/b/bar"))

	dt.commitPath("lib/a/b/bar")
	assert.True(t, dt.hasPath("lib/a"))
	assert.True(t, dt.hasPath("lib/a/b"))
	assert.True(t, dt.hasPath("lib/a/b/foo"))
	assert.True(t, dt.hasPath("lib/a/b/bar"))
	assert.False(t, dt.hasPath("lib/a/b/bar/baz"))
}

func Test_DirTreeListRelativeDirs(t *testing.T) {
	dt := NewDirTree("/usr")
	dt.CommitFiles([]string{
		"lib/a/b/c/hello.png",
	})

	assert.EqualValues(t, []string{
		"lib/a/b/c",
		"lib/a/b",
		"lib/a",
		"lib",
		".",
	}, dt.ListRelativeDirs())
}
