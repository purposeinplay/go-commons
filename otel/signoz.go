package otel

import (
	"context"
	"errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	otellog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"log"
)

func InitSigNozTracer(
	serviceName string,
	collectorURL string,
	logCollectorURL string,
	withTLS bool,
) func(context.Context) error {
	secureOption := otlptracegrpc.WithInsecure()

	if withTLS {
		secureOption = otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))

	}

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			secureOption,
			otlptracegrpc.WithEndpoint(collectorURL),
		),
	)

	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}

	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("library.language", "go"),
		),
	)
	if err != nil {
		log.Fatalf("Could not set resources: %v", err)
	}

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(resources),
		),
	)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{},
		),
	)

	conn, err := grpc.NewClient(
		logCollectorURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to create stdout exporter: %v", err)
	}

	exp, err := otlploggrpc.New(
		context.Background(),
		otlploggrpc.WithGRPCConn(conn),
	)
	if err != nil {
		log.Fatalf("Failed to create stdout exporter: %v", err)
	}

	provider := otellog.NewLoggerProvider(
		otellog.WithProcessor(otellog.NewBatchProcessor(exp)),
		otellog.WithResource(resources),
	)

	global.SetLoggerProvider(provider)

	return func(ctx context.Context) error {
		return errors.Join(
			provider.Shutdown(ctx),
			exporter.Shutdown(ctx),
			conn.Close(),
		)
	}
}
