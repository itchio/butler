package comm

import "github.com/itchio/wharf/pwr"

func NewStateConsumer() *pwr.StateConsumer {
	return &pwr.StateConsumer{
		OnProgress:      Progress,
		OnProgressLabel: ProgressLabel,
		OnMessage:       Logl,
	}
}
