package otel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ trace.TracerProvider = (*TracerProvider)(nil)

// TracerProvider is a wrapper around the GRPC OpenTelemetry TracerProvider.
type TracerProvider struct {
	embedded.TracerProvider

	conn *grpc.ClientConn
	tp   *sdktrace.TracerProvider
}

// Tracer returns a named tracer.
func (t *TracerProvider) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	return t.tp.Tracer(name, options...)
}

// Close closes the TracerProvider and the GRPC Conn.
func (t *TracerProvider) Close() error {
	var errs error

	const timeout = 10 * time.Second

	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := t.tp.Shutdown(timeoutCtx); err != nil {
		errs = errors.Join(errs, fmt.Errorf("shutdown: %w", err))
	}

	if err := t.conn.Close(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("close connection: %w", err))
	}

	return errs
}

// Init initializes the OpenTelemetry SDK with the OTLP exporter.
func Init(
	ctx context.Context,
	otelCollectorEndpoint string,
	serviceName string,
) (*TracerProvider, error) {
	oltpCollectorConn, err := newOTLPCollectorConn(otelCollectorEndpoint)
	if err != nil {
		return nil, fmt.Errorf("new otlp collector connection: %w", err)
	}

	tracerProvider, err := newTracerProvider(ctx, oltpCollectorConn, serviceName)
	if err != nil {
		return nil, fmt.Errorf("new trace provider: %w", err)
	}

	// report runtime metrics
	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second)); err != nil {
		return nil, fmt.Errorf("start runtime instrumentation: %w", err)
	}

	// Set the global TracerProvider.
	otel.SetTracerProvider(tracerProvider)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{},
		),
	)

	return &TracerProvider{
		tp:   tracerProvider,
		conn: oltpCollectorConn,
	}, nil
}

func newOTLPCollectorConn(
	otlpCollectorEndpoint string,
) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		otlpCollectorEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("create new grpc client: %w", err)
	}

	return conn, err
}

func newTracerProvider(
	ctx context.Context,
	otlpCollectorConn *grpc.ClientConn,
	serviceName string,
) (*sdktrace.TracerProvider, error) {
	var (
		exporter *otlptrace.Exporter
		res      *resource.Resource
	)

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		e, err := otlptracegrpc.New(
			egCtx,
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithGRPCConn(otlpCollectorConn),
		)
		if err != nil {
			return fmt.Errorf("create trace exporter: %w", err)
		}

		exporter = e

		return nil
	})

	eg.Go(func() error {
		// Create resource, which describes the service that will generate traces.
		r, err := resource.New(
			egCtx,
			resource.WithAttributes(
				// the service name used to display traces in backends
				semconv.ServiceNameKey.String(serviceName),
			),
		)
		if err != nil {
			return fmt.Errorf("create resource: %w", err)
		}

		res = r

		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	const alwaysSampleRatio = 1

	// Create a TracerProvider with the exporter.
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(
			sdktrace.ParentBased(
				sdktrace.TraceIDRatioBased(alwaysSampleRatio),
			),
		),
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(exporter)),
		sdktrace.WithResource(res),
	)

	return traceProvider, nil
}
