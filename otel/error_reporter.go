package otel

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"log/slog"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ErrorReporter provides structured error reporting with OpenTelemetry tracing integration.
// It combines structured logging with distributed tracing to capture and report errors
// in a way that's observable across service boundaries.
type ErrorReporter struct {
	// Logger is the structured logger used for error logging.
	// It should not be nil.
	Logger *slog.Logger

	// TraceProvider is the OpenTelemetry trace provider used to create new spans
	// when no active span exists in the context. Can be nil if tracing is disabled.
	TraceProvider trace.TracerProvider
}

// ReportError reports an error using both structured logging and OpenTelemetry tracing.
// 
// The function will:
//   - Always log the error using the structured logger
//   - Record the error in an OpenTelemetry span if tracing is available
//   - Create a new span if no active span exists but a TraceProvider is configured
//   - Set the span status to error and record stack trace information
//
// Parameters:
//   - ctx: Context that may contain an active OpenTelemetry span
//   - err: The error to report (must not be nil)
//
// The function is safe to call even when tracing is not configured.
func (r ErrorReporter) ReportError(ctx context.Context, err error) {
	if err == nil {
		return
	}

	if r.Logger == nil {
		// Use the default logger if the provided logger is nil.
		slog.Error("internal error: logger is nil, cannot report error", slog.Any("error", err))
		return
	}

	// Always log the error first
	r.Logger.ErrorContext(ctx, "internal error", slog.Any("error", err))

	// Handle tracing if available
	r.recordErrorInSpan(ctx, err)
}

// recordErrorInSpan handles the OpenTelemetry span operations for error reporting.
// It will either use an existing active span or create a new one if a TraceProvider is available.
func (r ErrorReporter) recordErrorInSpan(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)

	// Check if we have a valid active span
	if !span.SpanContext().IsValid() {
		// No valid span exists, try to create one if we have a TraceProvider
		if r.TraceProvider == nil {
			return // No tracing available
		}

		tracer := r.newTracer()

		_, span = tracer.Start(ctx, "error_reporting")

		// Ensure we end the span when we're done
		defer span.End()
	}

	// Record the error in the span with stack trace
	span.RecordError(err, trace.WithStackTrace(true))
	span.SetStatus(codes.Error, "internal error")

	// Add additional error context as span attributes
	span.SetAttributes(
		attribute.String("error.type", fmt.Sprintf("%T", err)),
		attribute.String("error.message", err.Error()),
	)
}

// scopeName defines the instrumentation scope for OpenTelemetry tracing.
// This should match your module/package name for proper trace attribution.
const scopeName = "github.com/purposeinplay/go-commons/otel"

// newTracer creates a new OpenTelemetry tracer using the configured TraceProvider.
// This is a helper method to encapsulate tracer creation with the correct scope.
func (r ErrorReporter) newTracer() trace.Tracer {
	return r.TraceProvider.Tracer(scopeName)
}

// Example usage:
//
//   func main() {
//       logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//       
//       // Initialize OpenTelemetry trace provider (implementation details omitted)
//       traceProvider := initTraceProvider()
//       
//       reporter := &otel.ErrorReporter{Logger: logger, TraceProvider: traceProvider}
//       
//       ctx := context.Background()
//       err = someOperation()
//       if err != nil {
//           reporter.ReportError(ctx, err)
//       }
//   }
