package comm

import "github.com/itchio/headway/state"

// NewStateConsumer returns an implementor of `pwr.StateConsumer` that prints
// directly to the console via butler's logging functions.
func NewStateConsumer() *state.Consumer {
	return &state.Consumer{
		OnProgress:       Progress,
		OnProgressLabel:  ProgressLabel,
		OnPauseProgress:  PauseProgress,
		OnResumeProgress: ResumeProgress,
		OnMessage:        Logl,
	}
}
