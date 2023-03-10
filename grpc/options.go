package grpc

import (
	"context"
	"net"

	"github.com/go-chi/chi/v5"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

// A ServerOption sets options such as credentials,
// codec and keepalive parameters, etc.
//
// The main purpose of Options in this package is to
// wrap the most common grpc parameter used in the company
// in order to provide the developers with a single point
// of entry for configuring different projects grpc servers.
type ServerOption interface {
	apply(*serverOptions)
}

// funcServerOption wraps a function that
// modifies serverOptions into an
// implementation of the ServerOption interface.
type funcServerOption struct {
	f func(*serverOptions)
}

func (fdo *funcServerOption) apply(do *serverOptions) {
	fdo.f(do)
}

func newFuncServerOption(f func(*serverOptions)) *funcServerOption {
	return &funcServerOption{
		f: f,
	}
}

type logging struct {
	logger         *zap.Logger
	ignoredMethods []string
}

type serverOptions struct {
	tracing                       bool
	gateway                       bool
	debugStandardLibraryEndpoints bool
	logging                       *logging
	address                       string
	grpcServerOptions             []grpc.ServerOption
	muxOptions                    []runtime.ServeMuxOption
	httpMiddlewares               chi.Middlewares
	registerServer                registerServerFunc
	registerGateway               registerGatewayFunc
	grpcListener                  net.Listener
	unaryServerInterceptors       []grpc.UnaryServerInterceptor
	errorHandler                  ErrorHandler
	panicHandler                  PanicHandler
	monitorOperationer            MonitorOperationer
	gatewayCorsOptions            cors.Options
}

// WithAddress configures the Server to listen to the given address
// in case of the Gateway server. And in case of the grpc server
// it uses the same address but port-1.
func WithAddress(a string) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.address = a
	})
}

// WithGRPCServerOptions configures the grpc server to use the given
// grpc server options.
func WithGRPCServerOptions(opts []grpc.ServerOption) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.grpcServerOptions = opts
	})
}

// WithMuxOptions configures the underlying runtime.ServeMux of the Gateway
// Server. The ServeMux as a handler for the http server.
func WithMuxOptions(opts []runtime.ServeMuxOption) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.muxOptions = opts
	})
}

// WithHTTPMiddlewares configures the Gateway Server http handler
// to use the provided middlewares.
func WithHTTPMiddlewares(mw chi.Middlewares) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.httpMiddlewares = mw
	})
}

// WithRegisterServerFunc registers a GRPC service to the
// GRPC server.
func WithRegisterServerFunc(f registerServerFunc) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.registerServer = f
	})
}

// WithRegisterGatewayFunc registers a GRPC service to the
// Gateway server.
func WithRegisterGatewayFunc(f registerGatewayFunc) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.registerGateway = f
	})
}

// WithGRPCListener configures the GRPC server to use the given
// net.Listener instead of configuring an address.
// ! Prefer to use this only in testing.
func WithGRPCListener(lis net.Listener) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.grpcListener = lis
		o.gateway = false
	})
}

// WithTracing enables tracing for both servers.
func WithTracing() ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.tracing = true
	})
}

// WithNoGateway disables the gateway server.
// ! Prefer to use this only in testing.
func WithNoGateway() ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.gateway = false
	})
}

// WithDebug enables logging for the servers.
func WithDebug(logger *zap.Logger, ignoredMethods ...string) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.logging = &logging{
			logger:         logger,
			ignoredMethods: ignoredMethods,
		}
	})
}

// WithUnaryServerInterceptorLogger adds an interceptor to the GRPC server
// that adds the given zap.Logger to the context.
func WithUnaryServerInterceptorLogger(logger *zap.Logger) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.unaryServerInterceptors = append(
			o.unaryServerInterceptors,
			grpc_zap.UnaryServerInterceptor(logger),
		)
	})
}

// WithUnaryServerInterceptorCodeGen adds an interceptor to the GRPC server
// that exports log fields from requests.
func WithUnaryServerInterceptorCodeGen() ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.unaryServerInterceptors = append(
			o.unaryServerInterceptors,
			grpc_ctxtags.UnaryServerInterceptor(
				grpc_ctxtags.WithFieldExtractor(
					grpc_ctxtags.CodeGenRequestFieldExtractor,
				),
			),
		)
	})
}

// WithUnaryServerInterceptorContextPropagation adds an interceptor to the
// GRPC server that binds the received metadata on the incoming context
// to the outgoing context.
func WithUnaryServerInterceptorContextPropagation() ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.unaryServerInterceptors = append(
			o.unaryServerInterceptors,
			func(
				ctx context.Context,
				req interface{},
				_ *grpc.UnaryServerInfo,
				handler grpc.UnaryHandler,
			) (interface{}, error) {
				outgoingCtx := ctx

				if md, ok := metadata.FromIncomingContext(ctx); ok {
					outgoingCtx = metadata.NewOutgoingContext(ctx, md)
				}

				return handler(outgoingCtx, req)
			},
		)
	})
}

// WithUnaryServerInterceptorAuthFunc adds an interceptor to the GRPC server
// that executes a per-request auth.
func WithUnaryServerInterceptorAuthFunc(
	authFunc grpc_auth.AuthFunc,
) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.unaryServerInterceptors = append(
			o.unaryServerInterceptors,
			grpc_auth.UnaryServerInterceptor(authFunc),
		)
	})
}

// WithMonitorOperationer adds an interceptor to the GRPC server
// that instruments the grpc handler.
func WithMonitorOperationer(
	monitorOperationer MonitorOperationer,
) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.monitorOperationer = monitorOperationer
	})
}

// WithPanicHandler adds an interceptor to the GRPC server
// that intercepts and handles panics.
func WithPanicHandler(panicHandler PanicHandler) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.panicHandler = panicHandler
	})
}

// WithErrorHandler adds an interceptor to the GRPC server
// that intercepts and handles the error returned by the handler.
func WithErrorHandler(errorHandler ErrorHandler) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.errorHandler = errorHandler
	})
}

// WithUnaryServerInterceptorRecovery adds an interceptor to the GRPC server
// that recovers from panics.
func WithUnaryServerInterceptorRecovery(
	recoveryHandler grpc_recovery.RecoveryHandlerFunc,
) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.unaryServerInterceptors = append(
			o.unaryServerInterceptors,
			grpc_recovery.UnaryServerInterceptor(
				grpc_recovery.WithRecoveryHandler(recoveryHandler),
			),
		)
	})
}

// WithUnaryServerInterceptor adds an interceptor to the GRPC server.
func WithUnaryServerInterceptor(
	unaryInterceptor grpc.UnaryServerInterceptor,
) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.unaryServerInterceptors = append(
			o.unaryServerInterceptors,
			unaryInterceptor,
		)
	})
}

// WithDebugStandardLibraryEndpoints registers the debug routes from
// the standard library to the gateway.
func WithDebugStandardLibraryEndpoints() ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.debugStandardLibraryEndpoints = true
	})
}

// WithGatewayCorsOptions sets the options to be used with CORS for the gateway server.
func WithGatewayCorsOptions(opts cors.Options) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.gatewayCorsOptions = opts
	})
}

func defaultServerOptions() serverOptions {
	return serverOptions{
		tracing:                       false,
		gateway:                       true,
		debugStandardLibraryEndpoints: false,
		logging:                       nil,
		address:                       "0.0.0.0:7350",
		grpcServerOptions:             nil,
		muxOptions: []runtime.ServeMuxOption{
			runtime.WithMarshalerOption(
				runtime.MIMEWildcard,
				&runtime.HTTPBodyMarshaler{
					Marshaler: &runtime.JSONPb{
						MarshalOptions: protojson.MarshalOptions{
							UseProtoNames:   true,
							UseEnumNumbers:  false,
							EmitUnpopulated: true,
						},
						UnmarshalOptions: protojson.UnmarshalOptions{
							DiscardUnknown: true,
						},
					},
				},
			),
		},
		httpMiddlewares:         nil,
		registerServer:          nil,
		registerGateway:         nil,
		grpcListener:            nil,
		unaryServerInterceptors: nil,
		errorHandler:            nil,
		panicHandler:            nil,
		monitorOperationer:      nil,
		gatewayCorsOptions: cors.Options{
			AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			ExposedHeaders:   []string{"Link", "X-Total-Count"},
			AllowCredentials: true,
		},
	}
}
