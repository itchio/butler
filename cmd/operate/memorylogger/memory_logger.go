package memorylogger

import "github.com/itchio/wharf/state"

// MemoryLogger

type MemoryLogger struct {
	items []*MemoryLogItem
}

type MemoryLogItem struct {
	level   string
	message string
}

func New() *MemoryLogger {
	return &MemoryLogger{}
}

func (ml *MemoryLogger) Consumer() *state.Consumer {
	return &state.Consumer{
		OnMessage: func(level string, message string) {
			ml.items = append(ml.items, &MemoryLogItem{level, message})
		},
	}
}

func (ml *MemoryLogger) Copy(dst *state.Consumer) {
	for _, item := range ml.items {
		dst.OnMessage(item.level, item.message)
	}
}
