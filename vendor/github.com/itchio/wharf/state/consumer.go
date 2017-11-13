package state

import (
	"fmt"

	"github.com/itchio/wharf/counter"
)

// ProgressCallback is called periodically to announce the degree of completeness of an operation
type ProgressCallback func(alpha float64)

// ProgressLabelCallback is called when the progress label should be changed
type ProgressLabelCallback func(label string)

// MessageCallback is called when a log message has to be printed
type MessageCallback func(level, msg string)

type VoidCallback func()

// Consumer holds callbacks for the various state changes one
// might want to consume (show progress to the user, store messages
// in a text file, etc.)
type Consumer struct {
	OnProgress       ProgressCallback
	OnPauseProgress  VoidCallback
	OnResumeProgress VoidCallback
	OnProgressLabel  ProgressLabelCallback
	OnMessage        MessageCallback
}

// Progress announces the degree of completion of a task, in the [0,1] interval
func (c *Consumer) Progress(progress float64) {
	if c.OnProgress != nil {
		c.OnProgress(progress)
	}
}

func (c *Consumer) PauseProgress() {
	if c.OnPauseProgress != nil {
		c.OnPauseProgress()
	}
}

func (c *Consumer) ResumeProgress() {
	if c.OnResumeProgress != nil {
		c.OnResumeProgress()
	}
}

// ProgressLabel gives extra info about which task is currently being executed
func (c *Consumer) ProgressLabel(label string) {
	if c.OnProgressLabel != nil {
		c.OnProgressLabel(label)
	}
}

// Debug logs debug-level messages
func (c *Consumer) Debug(msg string) {
	if c.OnMessage != nil {
		c.OnMessage("debug", msg)
	}
}

// Debugf is a formatted variant of Debug
func (c *Consumer) Debugf(msg string, args ...interface{}) {
	if c.OnMessage != nil {
		c.OnMessage("debug", fmt.Sprintf(msg, args...))
	}
}

// Info logs info-level messages
func (c *Consumer) Info(msg string) {
	if c.OnMessage != nil {
		c.OnMessage("info", msg)
	}
}

// Infof is a formatted variant of Info
func (c *Consumer) Infof(msg string, args ...interface{}) {
	if c.OnMessage != nil {
		c.OnMessage("info", fmt.Sprintf(msg, args...))
	}
}

// Warn logs warning-level messages
func (c *Consumer) Warn(msg string) {
	if c.OnMessage != nil {
		c.OnMessage("warning", msg)
	}
}

// Warnf is a formatted version of Warn
func (c *Consumer) Warnf(msg string, args ...interface{}) {
	if c.OnMessage != nil {
		c.OnMessage("warning", fmt.Sprintf(msg, args...))
	}
}

// CountCallback returns a function suitable for counter.NewWriterCallback
// or counter.NewReaderCallback
func (c *Consumer) CountCallback(totalSize int64) counter.CountCallback {
	return func(count int64) {
		c.Progress(float64(count) / float64(totalSize))
	}
}
