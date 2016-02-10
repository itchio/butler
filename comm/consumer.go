package comm

import "github.com/itchio/wharf/pwr"

func NewStateConsumer() *pwr.StateConsumer {
	return &pwr.StateConsumer{
		OnProgress: Progress,
		OnMessage:  Logl,
	}
}
