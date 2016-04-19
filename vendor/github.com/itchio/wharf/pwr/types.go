package pwr

import "fmt"

// ProgressCallback is called periodically to announce the degree of completeness of an operation
type ProgressCallback func(percent float64)

// ProgressLabelCallback is called when the progress label should be changed
type ProgressLabelCallback func(label string)

// MessageCallback is called when a log message has to be printed
type MessageCallback func(level, msg string)

type VoidCallback func()

// StateConsumer holds callbacks for the various state changes one
// might want to consume (show progress to the user, store messages
// in a text file, etc.)
type StateConsumer struct {
	OnProgress       ProgressCallback
	OnPauseProgress  VoidCallback
	OnResumeProgress VoidCallback
	OnProgressLabel  ProgressLabelCallback
	OnMessage        MessageCallback
}

// Progress announces the degree of completion of a task
func (sc *StateConsumer) Progress(percent float64) {
	if sc.OnProgress != nil {
		sc.OnProgress(percent)
	}
}

func (sc *StateConsumer) PauseProgress() {
	if sc.OnPauseProgress != nil {
		sc.OnPauseProgress()
	}
}

func (sc *StateConsumer) ResumeProgress() {
	if sc.OnResumeProgress != nil {
		sc.OnResumeProgress()
	}
}

// ProgressLabel gives extra info about which task is currently being executed
func (sc *StateConsumer) ProgressLabel(label string) {
	if sc.OnProgressLabel != nil {
		sc.OnProgressLabel(label)
	}
}

// Debug logs debug-level messages
func (sc *StateConsumer) Debug(msg string) {
	if sc.OnMessage != nil {
		sc.OnMessage("debug", msg)
	}
}

// Debugf is a formatted variant of Debug
func (sc *StateConsumer) Debugf(msg string, args ...interface{}) {
	if sc.OnMessage != nil {
		sc.OnMessage("debug", fmt.Sprintf(msg, args...))
	}
}

// Info logs info-level messages
func (sc *StateConsumer) Info(msg string) {
	if sc.OnMessage != nil {
		sc.OnMessage("info", msg)
	}
}

// Infof is a formatted variant of Info
func (sc *StateConsumer) Infof(msg string, args ...interface{}) {
	if sc.OnMessage != nil {
		sc.OnMessage("info", fmt.Sprintf(msg, args...))
	}
}

// Warn logs warning-level messages
func (sc *StateConsumer) Warn(msg string) {
	if sc.OnMessage != nil {
		sc.OnMessage("warning", msg)
	}
}

// Warnf is a formatted version of Warn
func (sc *StateConsumer) Warnf(msg string, args ...interface{}) {
	if sc.OnMessage != nil {
		sc.OnMessage("warning", fmt.Sprintf(msg, args...))
	}
}
