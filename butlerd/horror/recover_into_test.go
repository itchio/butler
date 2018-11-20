package horror_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/itchio/butler/butlerd/horror"
)

func ExampleRecoverInto() {
	f := func() (retErr error) {
		defer horror.RecoverInto(&retErr)
		panic("Oh no")
	}
	err := f()
	fmt.Printf("Returned from f: %+v", err)
}

func TestRecoverInto_String(t *testing.T) {
	assert := assert.New(t)
	f := func() (retErr error) {
		defer horror.RecoverInto(&retErr)
		panic("Oh no")
	}
	err := f()
	assert.Error(err, "panic: Oh no")
}

func TestRecoverInto_Error(t *testing.T) {
	assert := assert.New(t)
	f := func() (retErr error) {
		defer horror.RecoverInto(&retErr)
		panic(errors.New("Oh no!"))
	}
	err := f()
	assert.Error(err, "Oh no!")
}
