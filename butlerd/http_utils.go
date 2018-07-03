package butlerd

import (
	"log"
	"net/http"

	"github.com/pkg/errors"
)

type httpError struct {
	code  int
	cause error
}

func (he *httpError) Error() string {
	return he.cause.Error()
}

func HTTPError(code int, msg string, args ...interface{}) error {
	err := errors.Errorf(msg, args...)
	return &httpError{code: code, cause: err}
}

type CoolHandler func(w http.ResponseWriter, r *http.Request) error

func H(f CoolHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				if rErr, ok := r.(error); ok {
					log.Printf("Panic! %+v", errors.WithStack(rErr))
				} else {
					log.Printf("Panic! %+v", r)
				}
				http.Error(w, "Internal Error", 500)
			}
		}()

		err := f(w, r)
		if err != nil {
			log.Printf("%+v", err)
			if he, ok := errors.Cause(err).(*httpError); ok {
				http.Error(w, he.cause.Error(), he.code)
			} else {
				http.Error(w, err.Error(), 500)
			}
		}
	}
}
