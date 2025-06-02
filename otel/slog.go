package otel

import (
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"log/slog"
)

func NewSlogLogger(name string) *slog.Logger {
	return otelslog.NewLogger(
		name,
	)
}
