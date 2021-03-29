package middleware

import (
	"fmt"
	cmiddleware "github.com/go-chi/chi/middleware"
	"github.com/purposeinplay/go-commons/logs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"time"
)

func NewLoggerMiddleware(logger *zap.Logger) func(next http.Handler) http.Handler {
	return cmiddleware.RequestLogger(&structuredLogger{logger})
}

type structuredLogger struct {
	Logger *zap.Logger
}

func (l *structuredLogger) NewLogEntry(r *http.Request) cmiddleware.LogEntry {
	entry := &logs.StructuredLoggerEntry{Logger: l.Logger}

	fields := []zapcore.Field{zap.String("ts", time.Now().UTC().Format(time.RFC1123))}

	if reqID := cmiddleware.GetReqID(r.Context()); reqID != "" {
		fields = append(fields, zap.String("req.id", reqID))
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	fields = append(fields, []zapcore.Field{
		zap.String("http_scheme", scheme),
		zap.String("http_proto", r.Proto),
		zap.String("http_method", r.Method),
		zap.String("remote_addr", r.RemoteAddr),
		zap.String("user_agent", r.UserAgent()),
		zap.String("uri", fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)),
	}...)

	entry.Logger = l.Logger.With(fields...)

	entry.Logger.Info("request started")

	return entry
}
