package otel

import (
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

func NewSlogLogger(name string) *slog.Logger {
	return otelslog.NewLogger(
		name,
	)
}
