package test

import (
	"context"
	"net"
	"testing"

	commonsgrpc "github.com/purposeinplay/go-commons/grpc"
	"github.com/purposeinplay/go-commons/otel"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/test/bufconn"
	"github.com/purposeinplay/go-commons/grpc/grpcclient"
	"github.com/purposeinplay/go-commons/grpc/test_data/greetpb"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc"
	ootel "go.opentelemetry.io/otel"
)

func TestIntegration(t *testing.T) {
	req := require.New(t)

	ctx := context.Background()

	err := otel.Init(ctx, "localhost:4317", "test-service")
	req.NoError(err)

	const bufSize = 1024 * 1024

	lis := bufconn.Listen(bufSize)
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	grpcServer, err := commonsgrpc.NewServer(
		commonsgrpc.WithGRPCListener(lis),
		commonsgrpc.WithDebug(zap.NewExample(), true),
		commonsgrpc.WithOTEL(),
		commonsgrpc.WithRegisterServerFunc(func(server *grpc.Server) {
			greetpb.RegisterGreetServiceServer(server, &greeterService{
				greetFunc: func() error { return nil },
			})
		}),
	)
	req.NoError(err)

	go func() {
		if err := grpcServer.ListenAndServe(); err != nil {
			t.Logf("listen and serve error: %s", err)
		}
	}()

	t.Cleanup(func() {
		if err := grpcServer.Close(); err != nil {
			t.Logf("failed to close grpc server: %v", err)
		}
	})

	conn, err := grpcclient.NewConn(
		"bufnet",
		grpcclient.WithContextDialer(bufDialer),
		grpcclient.WithNoTLS(),
		grpcclient.WithOTEL(),
	)
	req.NoError(err)

	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Logf("close client conn: %s", err)
		}
	})

	greeterClient := greetpb.NewGreetServiceClient(conn)

	resp, err := greeterClient.Greet(ctx, &greetpb.GreetRequest{
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
	greetFunc func() error
}

func (s *greeterService) Greet(
	ctx context.Context,
	req *greetpb.GreetRequest,
) (*greetpb.GreetResponse, error) {
	tracer := ootel.Tracer("test")

	ctx, span := tracer.Start(ctx, "greet")
	defer span.End()

	if s.greetFunc != nil {
		err := s.greetFunc()
		if err != nil {
			return nil, err
		}
	}

	res := req.Greeting.FirstName + req.Greeting.LastName

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if len(md["custom"]) > 0 {
			res += md["custom"][0]
		}
	}

	return &greetpb.GreetResponse{
		Result: res,
	}, nil
}
