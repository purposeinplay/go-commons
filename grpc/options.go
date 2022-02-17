package grpc

import (
	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/purposeinplay/go-commons/logs"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"net"
)

// A ServerOption sets options such as credentials, codec and keepalive parameters, etc.
type ServerOption interface {
	apply(*serverOptions)
}

// funcServerOption wraps a function that modifies serverOptions into an
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

type serverOptions struct {
	tracing           bool
	gateway           bool
	address           string
	port              int
	logger            *zap.Logger
	grpcServerOptions []grpc.ServerOption
	muxOptions        []runtime.ServeMuxOption
	httpMiddleware    chi.Middlewares
	registerServer    func(server *grpc.Server)
	registerGateway   func(mux *runtime.ServeMux, dialOptions []grpc.DialOption)
	grpcListener      net.Listener
}

func WithAddress(a string) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.address = a
	})
}

func WithPort(a int) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.port = a
	})
}

func WithGRPCServerOptions(opts []grpc.ServerOption) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.grpcServerOptions = opts
	})
}

func WithMuxOptions(opts []runtime.ServeMuxOption) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.muxOptions = opts
	})
}

func WithHttpMiddleware(mw chi.Middlewares) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.httpMiddleware = mw
	})
}

func WithRegisterServer(f func(server *grpc.Server)) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.registerServer = f
	})
}

func WithRegisterGateway(f func(mux *runtime.ServeMux, dialOptions []grpc.DialOption)) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.registerGateway = f
	})
}

func WithReplaceLogger(l *zap.Logger) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.logger = l
	})
}

func WithGRPCListener(lis net.Listener) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.grpcListener = lis
	})
}

func WithTracing(tracing bool) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.tracing = tracing
	})
}

func WithNoGateway() ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.gateway = false
	})
}

func defaultServerOptions() (serverOptions, error) {
	logger, err := logs.NewLogger()
	if err != nil {
		return serverOptions{}, err
	}

	return serverOptions{
		tracing:        false,
		gateway:        true,
		address:        "0.0.0.0",
		port:           7350,
		httpMiddleware: nil,
		logger:         logger,
		muxOptions: []runtime.ServeMuxOption{
			runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.HTTPBodyMarshaler{
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
			}),
		},
	}, nil
}
