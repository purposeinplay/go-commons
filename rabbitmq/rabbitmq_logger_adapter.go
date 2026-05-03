package rabbitmq

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
)

var _ watermill.LoggerAdapter = (*SlogLogger)(nil)

type SlogLogger struct {
	log *slog.Logger

	trace,
	debug bool
}

func NewSlogLoggerAdapter(
	log *slog.Logger,
	trace bool,
	debug bool,
) *SlogLogger {
	return &SlogLogger{
		log:   log,
		trace: trace,
		debug: debug,
	}
}

func (l SlogLogger) Error(msg string, err error, fields watermill.LogFields) {
	l.log.Error(msg, append(map2attrs(fields), slog.Any("error", err))...)
}

func (l SlogLogger) Info(msg string, fields watermill.LogFields) {
	l.log.Info(msg, map2attrs(fields)...)
}

func (l SlogLogger) Debug(msg string, fields watermill.LogFields) {
	if !l.debug {
		return
	}

	l.log.Debug(msg, map2attrs(fields)...)
}

func (l SlogLogger) Trace(msg string, fields watermill.LogFields) {
	if !l.trace {
		return
	}

	l.log.Debug(msg, map2attrs(fields)...)
}

func (l SlogLogger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return SlogLogger{
		log:   l.log.With(map2attrs(fields)...),
		trace: l.trace,
		debug: l.debug,
	}
}

func map2attrs(m map[string]any) []any {
	attrs := make([]any, 0, len(m))

	for k, v := range m {
		attrs = append(attrs, slog.Any(k, v))
	}

	return attrs
}
