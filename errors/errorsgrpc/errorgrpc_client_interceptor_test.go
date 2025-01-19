package errorsgrpc_test

import (
	"context"
	"net"
	"testing"

	// nolint: staticcheck
	"github.com/golang/protobuf/proto"
	"github.com/purposeinplay/go-commons/errors"
	"github.com/purposeinplay/go-commons/errors/errorsgrpc"
	commonserr "github.com/purposeinplay/go-commons/errors/proto/commons/error/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
	t *testing.T
}

// SayHello implements helloworld.GreeterServer.
func (s *server) SayHello(context.Context, *pb.HelloRequest) (*pb.HelloReply, error) {
	sts := status.New(codes.NotFound, errors.ErrorTypeNotFound.String())

	sts, err := sts.WithDetails(proto.MessageV1(&commonserr.ErrorResponse{
		ErrorCode: 1,
		Message:   "not found",
	}))
	if err != nil {
		s.t.Fatalf("failed to add details to error: %s", err)
	}

	return nil, sts.Err()
}

func TestErrors(t *testing.T) {
	t.Parallel()

	req := require.New(t)

	const bufSize = 1024 * 1024

	var (
		lis       = bufconn.Listen(bufSize)
		bufDialer = func(context.Context, string) (net.Conn, error) { return lis.Dial() }
	)

	grpcServer := grpc.NewServer()

	pb.RegisterGreeterServer(grpcServer, &server{t: t})

	t.Cleanup(grpcServer.Stop)

	done := make(chan struct{}, 1)

	go func() {
		defer close(done)

		serveErr := grpcServer.Serve(lis)
		assert.NoError(t, serveErr)
	}()

	clientConn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(errorsgrpc.UnmarshalErrorUnaryClientInterceptor()),
	)
	req.NoError(err)

	t.Cleanup(func() { req.NoError(clientConn.Close()) })

	greeterClient := pb.NewGreeterClient(clientConn)

	ctx := context.Background()

	_, err = greeterClient.SayHello(ctx, &pb.HelloRequest{})

	var appErr *errors.Error

	req.ErrorAs(err, &appErr)

	req.Equal(
		&errors.Error{
			Type:    errors.ErrorTypeNotFound,
			Code:    "1",
			Message: "not found",
		},
		appErr,
	)

	grpcServer.Stop()

	<-done
}
