package grpcutils

import (
	"context"
	"errors"
	"google.golang.org/grpc"

	"google.golang.org/grpc/metadata"
)

// Request ID errors.
var (
	ErrMetadataNotFound    = errors.New("metadata not found")
	ErrRequestIDNotPresent = errors.New("request id not present")
)

const requestIDHeader = "x-request-id"

// GetRequestIDFromCtx returns the request id from the grpc context.
func GetRequestIDFromCtx(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", ErrMetadataNotFound
	}

	requestID := md.Get(requestIDHeader)

	if len(requestID) != 1 {
		return "", ErrRequestIDNotPresent
	}

	return requestID[0], nil
}

// AppendRequestIDCtx appends a random request id to the ctx.
func AppendRequestIDCtx(
	ctx context.Context,
	requestID string,
) context.Context {
	return metadata.AppendToOutgoingContext(
		ctx,
		requestIDHeader, requestID,
	)
}

// SetOutgoingRequestIDFromIncoming sets the outgoing request id to the
// value present in the incoming key.
func SetOutgoingRequestIDFromIncoming(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	requestID := md.Get(requestIDHeader)

	if len(requestID) != 1 {
		return ctx
	}

	return metadata.AppendToOutgoingContext(
		ctx,
		requestIDHeader, requestID[0],
	)
}

func PassRequestIDUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		return handler(
			SetOutgoingRequestIDFromIncoming(ctx),
			req,
		)
	}
}
