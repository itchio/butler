package archive

import (
	"fmt"
	"strings"
)

func List(params *ListParams) (*Contents, error) {
	return eachHandler("list", func(handler Handler) (*Contents, error) {
		return handler.List(params)
	})
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
