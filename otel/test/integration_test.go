package test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	commonsgrpc "github.com/purposeinplay/go-commons/grpc"
	"github.com/purposeinplay/go-commons/grpc/grpcclient"
	"github.com/purposeinplay/go-commons/grpc/test_data/greetpb"
	"github.com/purposeinplay/go-commons/otel"
	"github.com/purposeinplay/go-commons/otel/test/graph"
	"github.com/purposeinplay/go-commons/otel/test/graph/model"
	"github.com/ravilushqa/otelgqlgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/filters"
	ootel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.23.0"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/test/bufconn"
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
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
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

func TestGraphQLIntegration(t *testing.T) {
	req := require.New(t)

	ctx := context.Background()

	tp, err := otel.Init(
		ctx,
		"localhost:4317",
		semconv.ServiceName("test-service"),
	)
	req.NoError(err)

	t.Cleanup(func() {
		if err := tp.Close(); err != nil {
			t.Logf("close tracer provider: %s", err)
		}
	})

	s := graph.NewServer(nil)

	http.HandleFunc("POST /query", s.ServeHTTP)
	http.HandleFunc(
		"GET /playground",
		playground.Handler("GraphQL playground", "/query"),
	)

	t.Log("starting server")

	err = http.ListenAndServe(":8080", nil)
	req.ErrorIs(err, http.ErrServerClosed)
}

func TestHTTPIntegration(t *testing.T) {
	req := require.New(t)

	ctx := context.Background()

	tp, err := otel.Init(
		ctx,
		"localhost:4317",
		semconv.ServiceName("test-service"),
	)
	req.NoError(err)

	t.Cleanup(func() {
		if err := tp.Close(); err != nil {
			t.Logf("close tracer provider: %s", err)
		}
	})

	mux := http.NewServeMux()

	mux.HandleFunc("/hello", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("hello"))
	})

	t.Log("starting server")

	err = http.ListenAndServe(":8080", otelhttp.NewHandler(mux, "test-service"))
	req.ErrorIs(err, http.ErrServerClosed)
}

func TestGRPCIntegration(t *testing.T) {
	req := require.New(t)

	ctx := context.Background()

	tp, err := otel.Init(
		ctx,
		"localhost:4317",
		semconv.ServiceName("test-service"),
	)
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
		commonsgrpc.WithOTEL(
			otelgrpc.WithFilter(filters.Not(filters.MethodName("Greet"))),
		),
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
		grpcclient.WithOTEL(
			otelgrpc.WithFilter(func(info *stats.RPCTagInfo) bool {
				return !strings.Contains(info.FullMethodName, "Greet")
			}),
		),
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
		commonsgrpc.WithOTEL(
			otelgrpc.WithFilter(filters.Not(filters.MethodName("Greet"))),
		),
		commonsgrpc.WithUnaryServerInterceptorLogger(logger),
		commonsgrpc.WithRegisterServerFunc(func(server *grpc.Server) {
			greetpb.RegisterGreetServiceServer(
				server,
				&greeterService{
					greetFunc: func(
						ctx context.Context,
						req *greetpb.GreetRequest,
					) (*greetpb.GreetResponse, error) {
						t.Log("greet func server 2")

						ctxzap.Info(ctx, "zap log")

						return greeterClient1.Greet(ctx, req)
					},
				},
			)
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
		grpcclient.WithOTEL(
			otelgrpc.WithFilter(filters.Not(filters.MethodName("Greet"))),
		),
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

func TestGRPCGraphQLIntegration(t *testing.T) {
	req := require.New(t)

	ctx := context.Background()

	tp, err := otel.Init(
		ctx,
		"localhost:4317",
		semconv.ServiceName("test-service"),
	)
	req.NoError(err)

	t.Cleanup(func() { tp.Close() })

	const bufSize = 1024 * 1024

	lis1 := bufconn.Listen(bufSize)
	bufDialer1 := func(context.Context, string) (net.Conn, error) {
		return lis1.Dial()
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

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		err := grpcServer1.ListenAndServe()
		assert.NoError(t, err)
	}()

	conn1, err := grpcclient.NewConn(
		"bufnet",
		grpcclient.WithContextDialer(bufDialer1),
		grpcclient.WithNoTLS(),
		grpcclient.WithOTEL(),
	)
	req.NoError(err)

	t.Cleanup(func() { conn1.Close() })

	greeterClient1 := greetpb.NewGreetServiceClient(conn1)

	s := graph.NewServer(
		func(ctx context.Context, id string) (*model.User, error) {
			resp, err := greeterClient1.Greet(
				ctx,
				&greetpb.GreetRequest{
					Greeting: &greetpb.Greeting{
						FirstName: "test",
						LastName:  "otel",
					},
				},
			)
			if err != nil {
				return nil, fmt.Errorf("greet: %w", err)
			}

			return &model.User{
				ID:   id,
				Name: resp.Result,
			}, nil
		},
	)

	mux := http.NewServeMux()

	s.AroundOperations(func(
		ctx context.Context,
		next graphql.OperationHandler,
	) graphql.ResponseHandler {
		t.Logf("op: %+v", graphql.GetOperationContext(ctx).Operation.Name)

		oc := graphql.GetOperationContext(ctx)
		if oc == nil {
			t.Log("no operation context")
			return next(ctx)
		}

		return next(otelgqlgen.SetOperationName(ctx, oc.OperationName))
	})

	s.Use(otelgqlgen.Middleware(otelgqlgen.WithCreateSpanFromFields(
		func(*graphql.FieldContext) bool {
			return false
		})))

	mux.HandleFunc(
		"POST /query",
		s.ServeHTTP,
	)
	mux.HandleFunc(
		"GET /playground",
		playground.Handler("GraphQL playground", "/query"),
	)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	t.Log("starting server")

	wg.Add(1)

	go func() {
		defer wg.Done()

		err := server.ListenAndServe()
		assert.ErrorIs(t, err, http.ErrServerClosed)
	}()

	resp, err := http.Post(
		"http://localhost:8080/query",
		"application/json",
		strings.NewReader(
			`{"query": "query GetUserID { getUser(id: \"1\") { id name } }"}`,
		),
	)
	req.NoError(err)

	body, err := io.ReadAll(resp.Body)
	req.NoError(err)

	err = resp.Body.Close()
	req.NoError(err)

	req.JSONEq(
		`{"data":{"getUser":{"id":"1","name":"testotel"}}}`,
		string(body),
	)

	err = server.Shutdown(ctx)
	req.NoError(err)

	err = conn1.Close()
	req.NoError(err)

	err = grpcServer1.Close()
	req.NoError(err)

	err = tp.Close()
	req.NoError(err)

	wg.Wait()
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
