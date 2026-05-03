// Package gcpslog provides a slog.Handler that emits records in the
// JSON shape expected by Google Cloud Logging: "level" is renamed to
// "severity" with values DEBUG/INFO/WARNING/ERROR/CRITICAL, and "msg"
// is renamed to "message".
//
// Service name is added by the caller via slog.Logger.With("service",
// name); no special parameter is provided.
//
// Usage:
//
//	logger := slog.New(gcpslog.NewHandler(os.Stdout, nil)).With("service", "myservice")
package gcpslog

import (
	"io"
	"log/slog"
)

// NewHandler returns a slog.Handler that wraps slog.NewJSONHandler with
// GCP Cloud Logging field renames and severity mapping. Pass nil for
// opts to use slog defaults.
func NewHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	cfg := slog.HandlerOptions{}
	if opts != nil {
		cfg = *opts
	}

	prev := cfg.ReplaceAttr
	cfg.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		if prev != nil {
			a = prev(groups, a)
		}

		if len(groups) > 0 {
			return a
		}

		switch a.Key {
		case slog.LevelKey:
			lvl, ok := a.Value.Any().(slog.Level)
			if !ok {
				return a
			}
			return slog.String("severity", severityFor(lvl))
		case slog.MessageKey:
			return slog.String("message", a.Value.String())
		}
		return a
	}

	return slog.NewJSONHandler(w, &cfg)
}

func severityFor(l slog.Level) string {
	switch {
	case l < slog.LevelInfo:
		return "DEBUG"
	case l < slog.LevelWarn:
		return "INFO"
	case l < slog.LevelError:
		return "WARNING"
	case l < slog.LevelError+4:
		return "ERROR"
	default:
		return "CRITICAL"
	}
}
