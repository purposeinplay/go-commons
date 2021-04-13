package logs

import (
	"fmt"
	cmiddleware "github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type StructuredLoggerEntry struct {
	Logger *zap.Logger
}

func (l *StructuredLoggerEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	l.Logger = l.Logger.With(
		zap.Int("status", status),
		zap.Int("bytes_length", bytes),
		zap.Float64("duration_ms", float64(elapsed.Nanoseconds())/1000000.0),
	)

	l.Logger.Info("request complete")
}

func (l *StructuredLoggerEntry) Panic(v interface{}, stack []byte) {
	l.Logger = l.Logger.With(
		zap.String("stack", string(stack)),
		zap.String("panic", fmt.Sprintf("%+v", v)),
	)
}

// Helper methods used by the application to get the request-scoped
// logger entry and set additional fields between handlers.
//
// This is a useful pattern to use to set state on the entry as it
// passes through the handler chain, which at any point can be logged
// with a call to .Print(), .Info(), etc.
func GetLogEntry(r *http.Request) *zap.Logger {
	entry, _ := cmiddleware.GetLogEntry(r).(*StructuredLoggerEntry)

	if entry == nil {
		logger := NewLogger()
		return logger
	}

	return entry.Logger
}
