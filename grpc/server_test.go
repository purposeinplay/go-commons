package grpc_test

import (
	"context"
	"github.com/matryer/is"
	commonsgrpc "github.com/purposeinplay/go-commons/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"testing"
)

func TestBufnet(t *testing.T) {
	i := is.New(t)

	const bufSize = 1024 * 1024

	lis := bufconn.Listen(bufSize)
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	t.Cleanup(func() {
		err := lis.Close()
		i.NoErr(err)
	})

	s := commonsgrpc.NewServer(
		commonsgrpc.WithGRPCListener(lis),
		commonsgrpc.WithNoGateway(),
	)

	t.Cleanup(s.Stop)

	_, err := grpc.Dial(
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithInsecure(),
	)
	i.NoErr(err)
}
