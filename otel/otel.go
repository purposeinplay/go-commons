package otel

import (
	"context"
	"errors"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	semconv "go.opentelemetry.io/otel/semconv/v1.23.0"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	logembedded "go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	traceembedded "go.opentelemetry.io/otel/trace/embedded"
)

var (
	_ log.LoggerProvider   = (*TelemetryProvider)(nil)
	_ trace.TracerProvider = (*TelemetryProvider)(nil)
)

// TelemetryProvider is a wrapper around the GRPC OpenTelemetry TelemetryProvider.
type TelemetryProvider struct {
	traceembedded.TracerProvider
	logembedded.LoggerProvider

	tracerProvider *sdktrace.TracerProvider
	loggerProvider *sdklog.LoggerProvider
}

func BaseAttributes(serviceName, serviceNamespace, version string) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.ServiceName(serviceName),
		semconv.ServiceNamespace(serviceNamespace),
		semconv.ServiceVersion(version),
	}
}

// Init initializes the OpenTelemetry SDK with the OTLP exporter.
func Init(
	ctx context.Context,
	otelCollectorEndpoint string,
	attributes ...attribute.KeyValue,
) (*TelemetryProvider, error) {
	traceExporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(otelCollectorEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			attributes...,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	const alwaysSampleRatio = 1

	// Create a TelemetryProvider with the traceExporter.
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(
			sdktrace.ParentBased(
				sdktrace.TraceIDRatioBased(alwaysSampleRatio),
			),
		),
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
	)

	// report runtime metrics
	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second)); err != nil {
		return nil, fmt.Errorf("start runtime instrumentation: %w", err)
	}

	// Set the global TelemetryProvider.
	otel.SetTracerProvider(tracerProvider)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{},
		),
	)

	logExproter, err := otlploggrpc.New(
		context.Background(),
		otlploggrpc.WithEndpoint(otelCollectorEndpoint),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp log grpc exporter: %w", err)
	}

	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExproter)),
		sdklog.WithResource(res),
	)

	global.SetLoggerProvider(loggerProvider)

	return &TelemetryProvider{
		tracerProvider: tracerProvider,
		loggerProvider: loggerProvider,
	}, nil
}

// Tracer returns a named tracer.
func (t *TelemetryProvider) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	return t.tracerProvider.Tracer(name, options...)
}

// Logger creates and returns a named logger with optional configuration options.
func (t *TelemetryProvider) Logger(name string, opts ...log.LoggerOption) log.Logger {
	return t.loggerProvider.Logger(name, opts...)
}

// Close closes the TelemetryProvider and the GRPC Conn.
func (t *TelemetryProvider) Close() error {
	var errs error

	const timeout = 10 * time.Second

	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := t.tracerProvider.Shutdown(timeoutCtx); err != nil {
		errs = errors.Join(errs, fmt.Errorf("shutdown tracer provider: %w", err))
	}

	if err := t.loggerProvider.Shutdown(timeoutCtx); err != nil {
		errs = errors.Join(errs, fmt.Errorf("shutdown logger provider: %w", err))
	}

	return errs
}
