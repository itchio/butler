package mansion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_StripAPISubdomain(t *testing.T) {
	var res string
	var err error
	res, err = stripApiSubdomain("https://api.itch.io/")
	assert.NoError(t, err)
	assert.EqualValues(t, "https://itch.io/", res)

	res, err = stripApiSubdomain("http://api.localhost.com:8080/")
	assert.NoError(t, err)
	assert.EqualValues(t, "http://localhost.com:8080/", res)

	res, err = stripApiSubdomain("woops")
	assert.NoError(t, err)
	assert.EqualValues(t, "woops", res)

	_, err = stripApiSubdomain("# definitely @)#(*% not an URL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URL escape")
}
