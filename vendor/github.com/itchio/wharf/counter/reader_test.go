package counter

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func must(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}

func Test_Reader_Count(t *testing.T) {
	buf := bytes.NewReader([]byte{1, 2, 3, 4, 5, 6})
	cr := NewReader(buf)
	_, err := ioutil.ReadAll(cr)
	must(t, err)

	assert.Equal(t, cr.Count(), int64(6))
}

func Test_Reader_Nil(t *testing.T) {
	cr := NewReader(nil)
	buf := make([]byte, 6)
	n, err := cr.Read(buf)
	assert.NoError(t, err)

	assert.Equal(t, n, 6)
	assert.Equal(t, cr.Count(), int64(6))

	assert.NoError(t, cr.Close())
}

func Test_Reader_Callback(t *testing.T) {
	count := int64(-1)
	onRead := func(c int64) { count = c }

	cr := NewReaderCallback(onRead, nil)

	buf := make([]byte, 6)

	_, err := cr.Read(buf)
	must(t, err)
	assert.Equal(t, count, int64(6))

	_, err = cr.Read(buf)
	must(t, err)
	assert.Equal(t, count, int64(12))

	_, err = cr.Read(buf)
	must(t, err)
	assert.Equal(t, count, int64(18))
}
