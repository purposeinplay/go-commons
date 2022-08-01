package grpcutils_test

import (
	"context"
	"errors"
	"testing"

	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/grpc/grpcutils"
	"google.golang.org/grpc/metadata"
)

func TestRequestID(t *testing.T) {
	i := is.New(t)

	ctx := context.Background()

	_, err := grpcutils.GetRequestIDFromCtx(ctx)
	errors.Is(err, grpcutils.ErrMetadataNotFound)

	ctx = metadata.NewIncomingContext(
		ctx,
		metadata.New(map[string]string{}),
	)

	_, err = grpcutils.GetRequestIDFromCtx(ctx)
	errors.Is(err, grpcutils.ErrRequestIDNotPresent)

	ctx = grpcutils.AppendRequestIDCtx(ctx)

	md, ok := metadata.FromOutgoingContext(ctx)
	i.True(ok)

	token := md.Get("x-request-id")
	i.True(len(token) == 1)

	ctx = metadata.NewIncomingContext(
		ctx,
		metadata.New(map[string]string{"x-request-id": "test"}),
	)

	id, err := grpcutils.GetRequestIDFromCtx(ctx)
	i.NoErr(err)

	i.Equal("test", id)
}
