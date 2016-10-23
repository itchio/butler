package counter

import (
	"io/ioutil"
	"testing"

	"github.com/alecthomas/assert"
)

func Test_Writer_Count(t *testing.T) {
	cw := NewWriter(ioutil.Discard)
	buf := []byte{1, 2, 3, 4, 5, 6}

	for i := 0; i < 6; i++ {
		_, err := cw.Write(buf)
		must(t, err)
	}

	assert.Equal(t, cw.Count(), int64(36))
	assert.NoError(t, cw.Close())
}

func Test_Writer_Nil(t *testing.T) {
	cw := NewWriter(nil)
	buf := []byte{1, 2, 3, 4, 5, 6}

	for i := 0; i < 6; i++ {
		_, err := cw.Write(buf)
		must(t, err)
	}

	assert.Equal(t, cw.Count(), int64(36))
}

func Test_Writer_Callback(t *testing.T) {
	count := int64(-1)
	onWrite := func(c int64) { count = c }

	cw := NewWriterCallback(onWrite, nil)
	buf := []byte{1, 2, 3, 4, 5, 6}

	_, err := cw.Write(buf)
	must(t, err)
	assert.Equal(t, count, int64(6))

	_, err = cw.Write(buf)
	must(t, err)
	assert.Equal(t, count, int64(12))

	_, err = cw.Write(buf)
	must(t, err)
	assert.Equal(t, count, int64(18))

	_, err = cw.Write(buf)
	must(t, err)
	assert.Equal(t, count, int64(24))
}
