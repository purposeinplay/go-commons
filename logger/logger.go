package logger

import (
	"fmt"

	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewStackdriverDevelopment returns a new *zap.Logger that supports
// Google Stackdriver's structured logging.
// Logging is enabled at DebugLevel and above.
func NewStackdriverDevelopment(service string) (*zap.Logger, error) {
	return newLoggerFromConfig(zapdriver.NewDevelopmentConfig(), service)
}

// NewStackdriverProduction returns a new *zap.Logger that supports
// Google Stackdriver's structured logging.
// Logging is enabled at InfoLevel and above.
func NewStackdriverProduction(service string) (*zap.Logger, error) {
	return newLoggerFromConfig(zapdriver.NewProductionConfig(), service)
}

func newLoggerFromConfig(cfg zap.Config, service string) (*zap.Logger, error) {
	cfg.OutputPaths = []string{"stdout"}
	cfg.DisableStacktrace = true
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.InitialFields = map[string]interface{}{
		"service": service,
	}

	log, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("config build: %w", err)
	}

	return log, nil
}
