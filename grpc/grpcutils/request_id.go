package grpcutils

import (
	"context"
	"errors"

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

	token := md.Get(requestIDHeader)

	if len(token) != 1 {
		return "", ErrRequestIDNotPresent
	}

	return token[0], nil
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
