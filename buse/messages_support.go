package buse

type RequestMessage interface {
	Method() string
}
