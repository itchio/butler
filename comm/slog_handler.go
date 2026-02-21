package comm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

type slogHandler struct {
	level  slog.Leveler
	attrs  []slog.Attr
	groups []string
}

var _ slog.Handler = (*slogHandler)(nil)

// NewSlogHandler returns a slog.Handler that emits logs through comm.
func NewSlogHandler(level slog.Leveler) slog.Handler {
	if level == nil {
		level = slog.LevelInfo
	}

	return &slogHandler{
		level: level,
	}
}

func (h *slogHandler) Enabled(_ context.Context, level slog.Level) bool {
	if h.level == nil {
		return level >= slog.LevelInfo
	}
	return level >= h.level.Level()
}

func (h *slogHandler) Handle(_ context.Context, r slog.Record) error {
	obj := JsonMessage{
		"type":    "log",
		"time":    time.Now().UTC().Unix(),
		"level":   slogLevelToCommLevel(r.Level),
		"message": r.Message,
	}

	for _, attr := range h.attrs {
		addAttr(obj, h.groups, attr)
	}
	r.Attrs(func(attr slog.Attr) bool {
		addAttr(obj, h.groups, attr)
		return true
	})

	if JsonEnabled() {
		// Intentionally bypass comm debug filtering: if a logger is enabled at
		// debug level, records should be emitted in JSON mode.
		sendJSON(obj)
		return nil
	}

	level, _ := obj["level"].(string)
	if level == "" {
		level = "info"
	}
	Logl(level, fmt.Sprintf("%v", obj["message"]))
	return nil
}

func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nh := &slogHandler{
		level:  h.level,
		groups: append([]string{}, h.groups...),
		attrs:  append([]slog.Attr{}, h.attrs...),
	}
	nh.attrs = append(nh.attrs, attrs...)
	return nh
}

func (h *slogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	nh := &slogHandler{
		level:  h.level,
		groups: append([]string{}, h.groups...),
		attrs:  append([]slog.Attr{}, h.attrs...),
	}
	nh.groups = append(nh.groups, name)
	return nh
}

func addAttr(obj JsonMessage, groups []string, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return
	}

	if attr.Value.Kind() == slog.KindGroup {
		nextGroups := append([]string{}, groups...)
		if attr.Key != "" {
			nextGroups = append(nextGroups, attr.Key)
		}
		for _, groupAttr := range attr.Value.Group() {
			addAttr(obj, nextGroups, groupAttr)
		}
		return
	}

	if attr.Key == "" {
		return
	}

	keyParts := append(append([]string{}, groups...), attr.Key)
	key := strings.Join(keyParts, ".")
	obj[key] = slogValueToAny(attr.Value)
}

func slogValueToAny(v slog.Value) any {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindBool:
		return v.Bool()
	case slog.KindInt64:
		return v.Int64()
	case slog.KindUint64:
		return v.Uint64()
	case slog.KindFloat64:
		return v.Float64()
	case slog.KindDuration:
		return v.Duration()
	case slog.KindTime:
		return v.Time()
	case slog.KindAny:
		return v.Any()
	default:
		return v.Any()
	}
}

func slogLevelToCommLevel(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "error"
	case level >= slog.LevelWarn:
		return "warning"
	case level >= slog.LevelInfo:
		return "info"
	default:
		return "debug"
	}
}
