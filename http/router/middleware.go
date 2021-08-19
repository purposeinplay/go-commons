package router

import (
	"fmt"
	cmiddleware "github.com/go-chi/chi/middleware"
	commonshttp "github.com/purposeinplay/go-commons/http"
	"github.com/purposeinplay/go-commons/logs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"os"
	"runtime/debug"
	"time"
)

type Middleware func(http.Handler) http.Handler

func MiddlewareFunc(f func(w http.ResponseWriter, r *http.Request, next http.Handler)) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			f(w, r, next)
		})
	}
}

// Recoverer is a middleware that recovers from panics, logs the panic (and a
// backtrace), and returns a HTTP 500 (Internal Server Error) status if
// possible. Recoverer prints a request ID if one is provided.
func Recoverer() Middleware {
	return MiddlewareFunc(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		defer func() {
			if rvr := recover(); rvr != nil {
				logEntry, err := logs.GetLogEntry(r)

				switch {
					case err != nil:
						logEntry.Sugar().Panic(rvr, debug.Stack())
					case logEntry == nil:
						fallthrough
					default:
						fmt.Fprintf(os.Stderr, "Panic: %+v\n", rvr)
						debug.PrintStack()
				}


				err = &commonshttp.HTTPError{
					Code:    http.StatusInternalServerError,
					Message: http.StatusText(http.StatusInternalServerError),
				}
				commonshttp.HandleError(err, w, r)
			}
		}()


		next.ServeHTTP(w, r)
	})
}


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