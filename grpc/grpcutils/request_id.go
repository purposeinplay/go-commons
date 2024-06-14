package grpcutils

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"go.opentelemetry.io/otel/trace"
)

// Request ID errors.
var (
	ErrMetadataNotFound    = errors.New("metadata not found")
	ErrRequestIDNotPresent = errors.New("request id not present")
)

const requestIDHeader = "x-request-id"

// GetRequestIDFromCtx returns the request id from the grpc context.
func GetRequestIDFromCtx(ctx context.Context) (string, error) {
	requestID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()
	if requestID != (trace.TraceID{}).String() {
		return requestID, nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", ErrMetadataNotFound
	}

	requestIDs := md.Get(requestIDHeader)

	if len(requestIDs) != 1 {
		return "", ErrRequestIDNotPresent
	}

	return requestIDs[0], nil
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

// PassRequestIDUnaryInterceptor takes the incoming request-id and
// sets it as the outgoing request id.
func PassRequestIDUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		return handler(
			SetOutgoingRequestIDFromIncoming(ctx),
			req,
		)
	}
}
