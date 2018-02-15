package buse

import (
	"bufio"
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type objType struct {
	A, B string
}

func TestLFObjectCodecSuccess(t *testing.T) {
	codec := LFObjectCodec{}
	buff := bytes.NewBuffer([]byte{})

	inObj := objType{"A", "B"}
	assert.Nil(t, codec.WriteObject(buff, &inObj))

	outObj := objType{}
	assert.Nil(t, codec.ReadObject(bufio.NewReader(buff), &outObj))
	assert.True(t, inObj == outObj)
}

type limitWriter struct {
	*bytes.Buffer
	limit int
	sofar int
}

func (l *limitWriter) Write(p []byte) (n int, err error) {
	if l.sofar > l.limit {
		return 0, errors.New("Limit reached")
	}
	n, err = l.Buffer.Write(p)
	l.sofar += n
	if l.sofar > l.limit {
		return n, errors.New("Limit reached")
	}
	return
}

func TestLFObjectCodecWriteFailures(t *testing.T) {
	codec := LFObjectCodec{}
	buff := bytes.NewBuffer(make([]byte, 0, 0))
	limited := &limitWriter{
		buff,
		1,
		0,
	}
	inObj := objType{"A", "B"}

	assert.NotNil(t, codec.WriteObject(limited, &inObj))
	limited = &limitWriter{
		buff,
		17,
		0,
	}
	assert.NotNil(t, codec.WriteObject(limited, &inObj))

	assert.NotNil(t, codec.WriteObject(buff, make(chan int)))
}
