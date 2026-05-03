package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// NewStructuredLogger returns a chi-compatible request-logging middleware
// backed by stdlib log/slog. It emits one "request started" event per
// incoming request and one "request complete" event per response, both
// decorated with method, path, request ID, scheme, proto, remote addr,
// user agent, and full URI.
//
// Usage:
//
//	r := chi.NewRouter()
//	r.Use(http.NewStructuredLogger(slog.Default()))
//
// To plug in a custom slog.Handler:
//
//	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
//	r.Use(http.NewStructuredLogger(logger))
//
// The corresponding StructuredLoggerEntry can be retrieved per-request
// via GetLogEntry(r).
func NewStructuredLogger(
	logger *slog.Logger,
) func(next http.Handler) http.Handler {
	return chimiddleware.RequestLogger(&structuredLogger{logger: logger})
}

type structuredLogger struct {
	logger *slog.Logger
}

// NewLogEntry creates a new log entry with attributes that add context to
// help identify it. The returned entry is updated in-place across the
// Write / Panic callbacks chi makes during the request lifecycle.
func (l *structuredLogger) NewLogEntry(req *http.Request) chimiddleware.LogEntry {
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	attrs := []any{
		slog.String("component", "api"),
		slog.String("method", req.Method),
		slog.String("path", req.URL.Path),
		slog.String("http.scheme", scheme),
		slog.String("http.proto", req.Proto),
		slog.String("http.method", req.Method),
		slog.String("remote_addr", req.RemoteAddr),
		slog.String("user_agent", req.UserAgent()),
		slog.String("uri", fmt.Sprintf("%s://%s%s", scheme, req.Host, req.RequestURI)),
	}

	if reqID := chimiddleware.GetReqID(req.Context()); reqID != "" {
		attrs = append(attrs, slog.String("req.id", reqID))
	}

	entry := &StructuredLoggerEntry{Logger: l.logger.With(attrs...)}
	entry.Logger.Info("request started")

	return entry
}

// StructuredLoggerEntry is the per-request log entry chi mutates across
// the request lifecycle (start → write → panic).
type StructuredLoggerEntry struct {
	Logger *slog.Logger
}

// Write adds response context to the entry and emits "request complete".
func (l *StructuredLoggerEntry) Write(
	status, bytes int,
	_ http.Header,
	elapsed time.Duration,
	_ any,
) {
	elapsedMs := float64(elapsed.Nanoseconds()) / 1_000_000.0
	l.Logger = l.Logger.With(
		slog.Int("res.status", status),
		slog.Int("res.bytes_length", bytes),
		slog.Float64("res.elapsed_ms", elapsedMs),
	)
	l.Logger.Info("request complete")
}

// Panic adds panic context to the entry. chi calls this and Write (with
// status 500) so we don't emit a separate event here — the recoverer
// downstream emits the explicit panic log.
func (l *StructuredLoggerEntry) Panic(v any, stack []byte) {
	l.Logger = l.Logger.With(
		slog.String("stack", string(stack)),
		slog.String("panic", fmt.Sprintf("%+v", v)),
	)
}

// WithError annotates the entry with an error attribute and returns the
// updated logger.
func (l *StructuredLoggerEntry) WithError(err error) *slog.Logger {
	l.Logger = l.Logger.With(slog.Any("error", err))
	return l.Logger
}

// GetLogEntry returns the in-context StructuredLoggerEntry for a
// request, or nil if no structured logger middleware is on the chain.
// Callers can mutate the entry's Logger across the request lifecycle
// (e.g. entry.Logger = entry.Logger.With("user_id", uid)) and chi's
// Write/Panic callbacks will pick up the augmented logger.
func GetLogEntry(r *http.Request) *StructuredLoggerEntry {
	entry, ok := chimiddleware.GetLogEntry(r).(*StructuredLoggerEntry)
	if !ok {
		return nil
	}
	return entry
}
