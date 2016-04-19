package comm

import "github.com/itchio/wharf/pwr"

// NewStateConsumer returns an implementor of `pwr.StateConsumer` that prints
// directly to the console via butler's logging functions.
func NewStateConsumer() *pwr.StateConsumer {
	return &pwr.StateConsumer{
		OnProgress:       Progress,
		OnProgressLabel:  ProgressLabel,
		OnPauseProgress:  PauseProgress,
		OnResumeProgress: ResumeProgress,
		OnMessage:        Logl,
	}
}
