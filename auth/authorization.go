package auth

import (
	"context"

	"github.com/dgrijalva/jwt-go"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthorizerConfig is the configuration for the authorizer interceptor.
type AuthorizerConfig struct {
	Secret string
}

// AuthorizerInterceptor is a server interceptor to authenticate and authorize requests.
type AuthorizerInterceptor struct {
	logger     *zap.Logger
	jwtManager *JWTManager
	config     *AuthorizerConfig
}

// ProviderClaims is a custom JWT claims that contains some provider's information.
type ProviderClaims struct {
	jwt.StandardClaims
}

// CtxProviderClaimsKey is a context key for provider claims.
type CtxProviderClaimsKey struct{}

// NewAuthorizerInterceptor creates a new authorizer interceptor.
func NewAuthorizerInterceptor(
	logger *zap.Logger,
	jwtManager *JWTManager,
	config *AuthorizerConfig,
) *AuthorizerInterceptor {
	return &AuthorizerInterceptor{
		logger:     logger,
		jwtManager: jwtManager,
		config:     config,
	}
}

// Unary returns a server interceptor function to authenticate and authorize unary RPC.
func (i *AuthorizerInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		authorize, err := i.authorize(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}

		return handler(authorize, req)
	}
}

func (i *AuthorizerInterceptor) authorize(ctx context.Context, _ string) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		i.logger.Error("metadata was not provided")
		return ctx, status.Errorf(codes.Unauthenticated, "signature is required")
	}

	signature, ok := md["x-api-key"]
	if !ok {
		return ctx, status.Errorf(codes.Unauthenticated, "invalid signature")
	}

	var claims ProviderClaims

	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}

	_, err := p.ParseWithClaims(signature[0], &claims, func(*jwt.Token) (any, error) {
		return []byte(i.config.Secret), nil
	})
	if err != nil {
		// nolint: revive
		return nil, status.Errorf(codes.Unauthenticated, "you are not authorized to access this resource")
	}

	return context.WithValue(ctx, CtxProviderClaimsKey{}, claims), nil
}
