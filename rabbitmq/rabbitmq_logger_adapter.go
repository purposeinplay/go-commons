package rabbitmq

import (
	"github.com/ThreeDotsLabs/watermill"
	"go.uber.org/zap"
)

var _ watermill.LoggerAdapter = (*ZapLogger)(nil)

type ZapLogger struct {
	log *zap.Logger

	trace,
	debug bool
}

func NewZapLoggerAdapter(
	log *zap.Logger,
	trace bool,
	debug bool,
) *ZapLogger {
	return &ZapLogger{
		log:   log,
		trace: trace,
		debug: debug,
	}
}

func (l ZapLogger) Error(msg string, err error, fields watermill.LogFields) {
	l.log.Error(msg, append(map2fields(fields), zap.Error(err))...)
}

func (l ZapLogger) Info(msg string, fields watermill.LogFields) {
	l.log.Info(msg, map2fields(fields)...)
}

func (l ZapLogger) Debug(msg string, fields watermill.LogFields) {
	if !l.debug {
		return
	}

	l.log.Debug(msg, map2fields(fields)...)
}

func (l ZapLogger) Trace(msg string, fields watermill.LogFields) {
	if !l.trace {
		return
	}

	l.log.Debug(msg, map2fields(fields)...)
}

func (l ZapLogger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	newLogger := l.log

	for field, data := range fields {
		newLogger = l.log.With(zap.Any(field, data))
	}

	return ZapLogger{
		log: newLogger,
	}
}

func map2fields(m map[string]interface{}) []zap.Field {
	fields := make([]zap.Field, 0, len(m))

	for k, v := range m {
		fields = append(fields, zap.Any(k, v))
	}

	return fields
}
