package otel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

func TestLogLevel(t *testing.T) {
	req := require.New(t)
	ctx := t.Context()

	exp, err := stdoutlog.New()
	req.NoError(err)

	stdoutExporter, err := stdoutlog.New()
	req.NoError(err)

	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(stdoutExporter)),
	)

	t.Cleanup(func() {
		err = loggerProvider.Shutdown(context.Background())
		req.NoError(err)
	})

	global.SetLoggerProvider(loggerProvider)

	logger := NewSlogLogger("test")

	logger.InfoContext(ctx, "info")

	logger.DebugContext(ctx, "debug")
}
