package operate

import "github.com/itchio/wharf/state"

// memoryLogger

type memoryLogger struct {
	items []*memoryLogItem
}

type memoryLogItem struct {
	level   string
	message string
}

func (ml *memoryLogger) Consumer() *state.Consumer {
	return &state.Consumer{
		OnMessage: func(level string, message string) {
			ml.items = append(ml.items, &memoryLogItem{level, message})
		},
	}
}

func (ml *memoryLogger) Copy(dst *state.Consumer) {
	for _, item := range ml.items {
		dst.OnMessage(item.level, item.message)
	}
}
