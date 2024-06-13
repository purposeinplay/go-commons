package test

import (
	"context"
	"net"
	"testing"

	commonsgrpc "github.com/purposeinplay/go-commons/grpc"
	"github.com/purposeinplay/go-commons/otel"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
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

	conn, err := grpc.NewClient(
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	req.NoError(err)

	err = conn.Close()
	req.NoError(err)
}
