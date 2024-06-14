package test

import (
	"context"
	"net"
	"testing"
	"time"

	commonsgrpc "github.com/purposeinplay/go-commons/grpc"
	"github.com/purposeinplay/go-commons/grpc/grpcclient"
	"github.com/purposeinplay/go-commons/grpc/test_data/greetpb"
	"github.com/purposeinplay/go-commons/otel"
	"github.com/stretchr/testify/require"
	ootel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

func TestTracer(t *testing.T) {
	req := require.New(t)

	ctx := context.Background()

	conn, err := grpc.NewClient(
		"localhost:4317",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	req.NoError(err)

	exp, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithGRPCConn(conn),
	)
	req.NoError(err)

	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := exp.Shutdown(shutdownCtx); err != nil {
			t.Logf("shutdown exporter: %s", err)
		}
	})

	ssp := sdktrace.NewBatchSpanProcessor(
		exp,
	)

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL)),
		sdktrace.WithSpanProcessor(ssp),
	)

	ootel.SetTracerProvider(tracerProvider)

	tracer := ootel.Tracer("test")

	_, sp := tracer.Start(ctx, "test")

	sp.SetAttributes(attribute.String("test", "test"))
	sp.End()
}

func TestIntegration(t *testing.T) {
	req := require.New(t)

	ctx := context.Background()

	tp, err := otel.Init(ctx, "localhost:4317", "test-service")
	req.NoError(err)

	t.Cleanup(func() {
		if err := tp.Close(); err != nil {
			t.Logf("close tracer provider: %s", err)
		}
	})

	const bufSize = 1024 * 1024

	lis1 := bufconn.Listen(bufSize)
	bufDialer1 := func(context.Context, string) (net.Conn, error) {
		return lis1.Dial()
	}

	lis2 := bufconn.Listen(bufSize)
	bufDialer2 := func(context.Context, string) (net.Conn, error) {
		return lis2.Dial()
	}

	grpcServer1, err := commonsgrpc.NewServer(
		commonsgrpc.WithGRPCListener(lis1),
		commonsgrpc.WithDebug(zap.NewExample(), true),
		commonsgrpc.WithOTEL(),
		commonsgrpc.WithRegisterServerFunc(func(server *grpc.Server) {
			greetpb.RegisterGreetServiceServer(server, &greeterService{})
		}),
	)
	req.NoError(err)

	go func() {
		if err := grpcServer1.ListenAndServe(); err != nil {
			t.Logf("listen and serve error: %s", err)
		}
	}()

	t.Cleanup(func() {
		if err := grpcServer1.Close(); err != nil {
			t.Logf("failed to close grpc server1: %v", err)
		}
	})

	conn1, err := grpcclient.NewConn(
		"bufnet",
		grpcclient.WithContextDialer(bufDialer1),
		grpcclient.WithNoTLS(),
		grpcclient.WithOTEL(),
	)
	req.NoError(err)

	t.Cleanup(func() {
		if err := conn1.Close(); err != nil {
			t.Logf("close client conn1: %s", err)
		}
	})

	logger, err := zap.NewDevelopment()
	req.NoError(err)

	greeterClient1 := greetpb.NewGreetServiceClient(conn1)

	grpcServer2, err := commonsgrpc.NewServer(
		commonsgrpc.WithGRPCListener(lis2),
		commonsgrpc.WithDebug(zap.NewExample(), true),
		commonsgrpc.WithOTEL(),
		commonsgrpc.WithUnaryServerInterceptorLogger(logger),
		commonsgrpc.WithRegisterServerFunc(func(server *grpc.Server) {
			greetpb.RegisterGreetServiceServer(server, &greeterService{
				greetFunc: func(ctx context.Context, req *greetpb.GreetRequest) (*greetpb.GreetResponse, error) {
					t.Log("greet func server 2")

					ctxzap.Info(ctx, "zap log")

					return greeterClient1.Greet(ctx, req)
				},
			})
		}),
	)
	req.NoError(err)

	go func() {
		if err := grpcServer2.ListenAndServe(); err != nil {
			t.Logf("listen and serve error: %s", err)
		}
	}()

	t.Cleanup(func() {
		if err := grpcServer2.Close(); err != nil {
			t.Logf("failed to close grpc server1: %v", err)
		}
	})

	conn2, err := grpcclient.NewConn(
		"bufnet",
		grpcclient.WithContextDialer(bufDialer2),
		grpcclient.WithNoTLS(),
		grpcclient.WithOTEL(),
	)
	req.NoError(err)

	t.Cleanup(func() {
		if err := conn2.Close(); err != nil {
			t.Logf("close client conn1: %s", err)
		}
	})

	greeterClient2 := greetpb.NewGreetServiceClient(conn2)

	resp, err := greeterClient2.Greet(ctx, &greetpb.GreetRequest{
		Greeting: &greetpb.Greeting{
			FirstName: "test",
			LastName:  "otel",
		},
	})
	req.NoError(err)

	t.Log(resp)
}

var _ greetpb.GreetServiceServer = (*greeterService)(nil)

type greeterService struct {
	greetpb.UnimplementedGreetServiceServer
	greetFunc func(context.Context, *greetpb.GreetRequest) (*greetpb.GreetResponse, error)
}

func (s *greeterService) Greet(
	ctx context.Context,
	req *greetpb.GreetRequest,
) (*greetpb.GreetResponse, error) {
	if s.greetFunc != nil {
		return s.greetFunc(ctx, req)
	}

	res := req.Greeting.FirstName + req.Greeting.LastName

	return &greetpb.GreetResponse{
		Result: res,
	}, nil
}
