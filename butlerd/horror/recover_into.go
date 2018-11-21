package horror

import (
	"fmt"

	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

// RecoverInto is a function you can defer-call at
// the start of a function that you know has a risk of
// panicking.
func RecoverInto(errp *error) {
	if r := recover(); r != nil {
		if rErr, ok := r.(error); ok {
			*errp = errors.WithStack(rErr)
		} else {
			*errp = errors.New(fmt.Sprintf("panic: %+v", r))
		}
	}
}

func RecoverAndLog(consumer *state.Consumer) {
	if r := recover(); r != nil {
		consumer.Errorf("Recovered panic: %+v", errors.Errorf("%+v", r))
	}
}
