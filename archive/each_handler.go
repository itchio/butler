package archive

import (
	"fmt"
	"strings"
)

func TryOpen(params *TryOpenParams) (Handler, error) {
	for _, h := range handlers {
		err := h.TryOpen(params)
		if err != nil {
			continue
		} else {
			return h, nil
		}
	}

	return nil, ErrUnrecognizedArchiveType
}

func Extract(params *ExtractParams) (*Contents, error) {
	return eachHandler("extract", func(handler Handler) (*Contents, error) {
		return handler.Extract(params)
	})
}

type EachHandlerFunc func(h Handler) (*Contents, error)

func eachHandler(op string, cb EachHandlerFunc) (*Contents, error) {
	var msgs []string

	for _, h := range handlers {
		res, err := cb(h)
		if err == nil {
			return res, nil
		}

		msgs = append(msgs, err.Error())
	}

	return nil, fmt.Errorf("no backend could %s: %s", op, strings.Join(msgs, " ; "))
}

func GetHandler(name string) Handler {
	for _, h := range handlers {
		if h.Name() == name {
			return h
		}
	}

	return nil
}
