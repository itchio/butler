package comm

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestSlogHandler_EmitsDebugInJSONModeWithoutVerbose(t *testing.T) {
	output := captureJSONLogs(t, func() {
		logger := slog.New(NewSlogHandler(slog.LevelDebug))
		logger.Debug("hades query",
			slog.String("query", "SELECT 1"),
			slog.Any("args", []any{1, "foo"}),
			slog.Duration("duration", 2500*time.Nanosecond),
		)
	})

	if len(output) != 1 {
		t.Fatalf("expected 1 log line, got %d", len(output))
	}
	logObj := output[0]

	if got, _ := logObj["type"].(string); got != "log" {
		t.Fatalf("expected type=log, got %#v", logObj["type"])
	}
	if got, _ := logObj["level"].(string); got != "debug" {
		t.Fatalf("expected level=debug, got %#v", logObj["level"])
	}
	if got, _ := logObj["message"].(string); got != "hades query" {
		t.Fatalf("expected message=hades query, got %#v", logObj["message"])
	}
	if got, _ := logObj["query"].(string); got != "SELECT 1" {
		t.Fatalf("expected query=SELECT 1, got %#v", logObj["query"])
	}
	if _, ok := logObj["time"]; !ok {
		t.Fatalf("expected time field")
	}
	if _, ok := logObj["duration"]; !ok {
		t.Fatalf("expected duration field")
	}
	args, ok := logObj["args"].([]any)
	if !ok || len(args) != 2 {
		t.Fatalf("expected args array with 2 values, got %#v", logObj["args"])
	}
}

func TestSlogHandler_WithAttrsAndGroup(t *testing.T) {
	output := captureJSONLogs(t, func() {
		logger := slog.New(NewSlogHandler(slog.LevelDebug)).
			WithGroup("db").
			With("component", "hades")
		logger.Debug("query", slog.String("sql", "PRAGMA foreign_keys = 0"))
	})

	if len(output) != 1 {
		t.Fatalf("expected 1 log line, got %d", len(output))
	}
	logObj := output[0]

	if got, _ := logObj["db.component"].(string); got != "hades" {
		t.Fatalf("expected db.component=hades, got %#v", logObj["db.component"])
	}
	if got, _ := logObj["db.sql"].(string); got != "PRAGMA foreign_keys = 0" {
		t.Fatalf("expected db.sql attr, got %#v", logObj["db.sql"])
	}
}

func TestSlogHandler_AllowsReservedKeyOverwrite(t *testing.T) {
	output := captureJSONLogs(t, func() {
		logger := slog.New(NewSlogHandler(slog.LevelDebug))
		logger.Debug("original",
			slog.String("message", "override"),
			slog.String("type", "custom"),
			slog.String("level", "weird"),
			slog.Int64("time", 123),
		)
	})

	if len(output) != 1 {
		t.Fatalf("expected 1 log line, got %d", len(output))
	}
	logObj := output[0]

	if got, _ := logObj["message"].(string); got != "override" {
		t.Fatalf("expected message override, got %#v", logObj["message"])
	}
	if got, _ := logObj["type"].(string); got != "custom" {
		t.Fatalf("expected type override, got %#v", logObj["type"])
	}
	if got, _ := logObj["level"].(string); got != "weird" {
		t.Fatalf("expected level override, got %#v", logObj["level"])
	}
	if got, _ := logObj["time"].(float64); got != 123 {
		t.Fatalf("expected time override, got %#v", logObj["time"])
	}
}

func captureJSONLogs(t *testing.T, fn func()) []map[string]any {
	t.Helper()

	oldSettings := *settings
	defer func() {
		*settings = oldSettings
	}()
	Configure(false, false, false, true, false, false, false)

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("closing writer: %v", err)
	}

	outBytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	var output []map[string]any
	for _, line := range bytes.Split(bytes.TrimSpace(outBytes), []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal(line, &obj); err != nil {
			t.Fatalf("unmarshal json line %q: %v", string(line), err)
		}
		output = append(output, obj)
	}

	return output
}
