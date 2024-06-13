package otel

import (
	"context"
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
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Init initializes the OpenTelemetry SDK with the OTLP exporter.
func Init(
	ctx context.Context,
	otelCollectorEndopoint string,
	serviceName string,
) error {
	oltpCollectorConn, err := newOTLPCollectorConn(otelCollectorEndopoint)
	if err != nil {
		return fmt.Errorf("failed to create otlp collector connection: %w", err)
	}

	traceProvider, err := newTraceProvider(ctx, oltpCollectorConn, serviceName)
	if err != nil {
		return fmt.Errorf("failed to create trace provider: %w", err)
	}

	// Start runtime instrumentation
	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second)); err != nil {
		return fmt.Errorf("failed to start runtime instrumentation: %w", err)
	}

	// Set the global TracerProvider.
	otel.SetTracerProvider(traceProvider)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{},
		),
	)

	return nil
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

func newTraceProvider(
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
