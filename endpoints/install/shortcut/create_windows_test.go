//go:build windows
// +build windows

package shortcut

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SanitizeFileName(t *testing.T) {
	assert := assert.New(t)

	assert.EqualValues("Super Game", sanitizeFileName("<Super Game>"))
	assert.EqualValues("Jedi Mindset The Revenge", sanitizeFileName("Jedi Mindset: The Revenge"))
	assert.EqualValues("Many Spaces", sanitizeFileName("Many     Spaces"))

	assert.EqualValues("CON_", sanitizeFileName("CON"))
	assert.EqualValues("con_", sanitizeFileName("con"))
}
