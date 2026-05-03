package gcpslog_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/purposeinplay/go-commons/gcpslog"
)

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, buf.String())
	}
	return got
}

func TestSeverityMapping(t *testing.T) {
	cases := []struct {
		level    slog.Level
		severity string
	}{
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelDebug + 1, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelInfo + 1, "INFO"},
		{slog.LevelWarn, "WARNING"},
		{slog.LevelError, "ERROR"},
		{slog.LevelError + 4, "CRITICAL"},
		{slog.LevelError + 100, "CRITICAL"},
	}

	for _, tc := range cases {
		t.Run(tc.severity, func(t *testing.T) {
			var buf bytes.Buffer
			h := gcpslog.NewHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
			logger := slog.New(h)
			logger.Log(t.Context(), tc.level, "hello")

			got := decode(t, &buf)
			if got["severity"] != tc.severity {
				t.Fatalf("severity: want %q, got %v", tc.severity, got["severity"])
			}
			if _, ok := got["level"]; ok {
				t.Fatalf("level key should be removed, got: %v", got["level"])
			}
		})
	}
}

func TestMessageRename(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(gcpslog.NewHandler(&buf, nil))
	logger.Info("hello world", "k", "v")

	got := decode(t, &buf)
	if got["message"] != "hello world" {
		t.Fatalf("message: want %q, got %v", "hello world", got["message"])
	}
	if _, ok := got["msg"]; ok {
		t.Fatalf("msg key should be removed")
	}
	if got["k"] != "v" {
		t.Fatalf("attr lost: %v", got)
	}
}

func TestServiceAttrViaWith(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(gcpslog.NewHandler(&buf, nil)).With("service", "svc-x")
	logger.Info("e")

	got := decode(t, &buf)
	if got["service"] != "svc-x" {
		t.Fatalf("service: want %q, got %v", "svc-x", got["service"])
	}
}

func TestPreservesUserReplaceAttr(t *testing.T) {
	var buf bytes.Buffer
	called := false
	opts := &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			called = true
			if a.Key == "secret" {
				return slog.String("secret", "***")
			}
			return a
		},
	}
	logger := slog.New(gcpslog.NewHandler(&buf, opts))
	logger.Info("e", "secret", "shh")

	got := decode(t, &buf)
	if !called {
		t.Fatal("user ReplaceAttr was not invoked")
	}
	if got["secret"] != "***" {
		t.Fatalf("user ReplaceAttr did not run: %v", got)
	}
	if got["severity"] != "INFO" {
		t.Fatalf("severity rename did not run after user ReplaceAttr: %v", got)
	}
}

func TestNestedGroupsUntouched(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(gcpslog.NewHandler(&buf, nil))
	logger.Info("e", slog.Group("g", slog.String("level", "raw"), slog.String("msg", "raw")))

	got := decode(t, &buf)
	g, ok := got["g"].(map[string]any)
	if !ok {
		t.Fatalf("group missing: %v", got)
	}
	if g["level"] != "raw" || g["msg"] != "raw" {
		t.Fatalf("nested group keys were rewritten: %v", g)
	}
}

func TestTimestampPresent(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(gcpslog.NewHandler(&buf, nil))
	logger.Info("e")

	got := decode(t, &buf)
	ts, ok := got["time"].(string)
	if !ok {
		t.Fatalf("time attr missing or wrong type: %v", got)
	}
	if _, err := time.Parse(time.RFC3339Nano, ts); err != nil {
		t.Fatalf("time not parseable: %v (raw: %s)", err, ts)
	}
}
