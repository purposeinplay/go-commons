package auth

import (
	"context"
	"log"

	"go.uber.org/zap"

	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/status"

	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc"
)

type AuthInterceptor struct {
	logger     *zap.Logger
	jwtManager *JWTManager
	authRoles  map[string][]string
}

func NewAuthInterceptor(logger *zap.Logger, jwtManager *JWTManager, authRoles map[string][]string) *AuthInterceptor {
	return &AuthInterceptor{
		logger:     logger,
		jwtManager: jwtManager,
		authRoles:  authRoles,
	}
}

// Unary returns a server interceptor function to authenticate and authorize unary RPC
func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		log.Println("--> unary interceptor: ", info.FullMethod)

		err := i.authorize(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

// Stream returns a server interceptor function to authenticate and authorize stream RPC
func (i *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		log.Println("--> stream interceptor: ", info.FullMethod)

		err := i.authorize(stream.Context(), info.FullMethod)
		if err != nil {
			return err
		}

		return handler(srv, stream)
	}
}

func (i *AuthInterceptor) authorize(ctx context.Context, method string) error {
	authRoles, ok := i.authRoles[method]
	if !ok {
		// public route
		return nil
	}

	md, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		i.logger.Error("metadata was not provided")
		return status.Errorf(codes.Unauthenticated, "auth token is invalid")
	}

	signature, err := ExtractTokenFromMetadata(md)

	if err != nil {
		i.logger.Error("ExtractTokenFromMetadata error", zap.Error(err))
		return status.Errorf(codes.Unauthenticated, "auth token is invalid")
	}

	claims, err := i.jwtManager.Verify(signature)

	if err != nil {
		return status.Errorf(codes.Unauthenticated, "auth token is invalid")
	}

	for _, role := range authRoles {
		if role == claims.Role {
			return nil
		}
	}

	return status.Error(codes.Unauthenticated, "no permission to access this RPC")
}
