package kafka

import (
	"github.com/ThreeDotsLabs/watermill"
	"go.uber.org/zap"
)

var _ watermill.LoggerAdapter = (*zapLogger)(nil)

type zapLogger struct {
	log *zap.Logger

	trace,
	debug bool
}

func newLoggerAdapter(
	log *zap.Logger,
) *zapLogger {
	return &zapLogger{
		log:   log,
		trace: false,
		debug: false,
	}
}

func (l zapLogger) Error(msg string, err error, fields watermill.LogFields) {
	l.log.Error(msg, append(map2fields(fields), zap.Error(err))...)
}

func (l zapLogger) Info(msg string, fields watermill.LogFields) {
	l.log.Info(msg, map2fields(fields)...)
}

func (l zapLogger) Debug(msg string, fields watermill.LogFields) {
	if !l.debug {
		return
	}

	l.log.Debug(msg, map2fields(fields)...)
}

func (l zapLogger) Trace(msg string, fields watermill.LogFields) {
	if !l.trace {
		return
	}

	l.log.Debug(msg, map2fields(fields)...)
}

func (l zapLogger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	newLogger := l.log

	for field, data := range fields {
		newLogger = l.log.With(zap.Any(field, data))
	}

	return zapLogger{
		log: newLogger,
	}
}

func map2fields(m map[string]any) []zap.Field {
	fields := make([]zap.Field, 0, len(m))

	for k, v := range m {
		fields = append(fields, zap.Any(k, v))
	}

	return fields
}
