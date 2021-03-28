package grpc

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/purposeinplay/go-commons/logs"

	"github.com/rs/cors"

	"github.com/go-chi/chi"

	"google.golang.org/protobuf/encoding/protojson"

	grpcgw "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"go.uber.org/zap"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

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
	address         string
	port            int
	logger          *zap.Logger
	serverOptions   []grpc.ServerOption
	httpMiddleware  chi.Middlewares
	registerServer  func(server *grpc.Server)
	registerGateway func(mux *runtime.ServeMux, dialOptions []grpc.DialOption)
}

func Address(a string) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.address = a
	})
}

func ServerOptions(opts []grpc.ServerOption) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.serverOptions = opts
	})
}

func HttpMiddleware(mw chi.Middlewares) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.httpMiddleware = mw
	})
}

func RegisterServer(f func(server *grpc.Server)) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.registerServer = f
	})
}

func RegisterGateway(f func(mux *runtime.ServeMux, dialOptions []grpc.DialOption)) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.registerGateway = f
	})
}

func ReplaceLogger(l *zap.Logger) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.logger = l
	})
}

var defaultServerOptions = serverOptions{
	address:        "0.0.0.0",
	port:           7350,
	httpMiddleware: nil,
	logger:         logs.NewLogger(),
}

// A ServerOption sets options such as credentials, codec and keepalive parameters, etc.
type ServerOption interface {
	apply(*serverOptions)
}

type Server struct {
	grpcServer        *grpc.Server
	GrpcGatewayServer *http.Server
	opts              serverOptions
}

func NewServer(opt ...ServerOption) *Server {

	opts := defaultServerOptions
	for _, o := range opt {
		o.apply(&opts)
	}

	grpcServer := grpc.NewServer(opts.serverOptions...)

	server := &Server{
		opts: opts,
		//logger:     logger,
		grpcServer: grpcServer,
	}

	reflection.Register(grpcServer)
	opts.registerServer(grpcServer)

	// Register and start GRPC server.
	opts.logger.Info("Starting gRPC server", zap.Int("port", opts.port-1))
	go func() {
		listen, err := net.Listen("tcp", fmt.Sprintf("%v:%v", opts.address, opts.port-1))
		if err != nil {
			opts.logger.Fatal("gRPC server listener failed to start", zap.Error(err))
		}

		err = grpcServer.Serve(listen)
		if err != nil {
			opts.logger.Fatal("gRPC server listener failed", zap.Error(err))
		}
	}()

	// Register and start GRPC Gateway server.
	dialAddr := fmt.Sprintf("127.0.0.1:%d", opts.port-1)
	if opts.address != "" {
		dialAddr = fmt.Sprintf("%v:%d", opts.address, opts.port-1)
	}
	opts.logger.Info("Starting gRPC gateway for HTTP requests", zap.String("gRPC gateway", dialAddr))
	go func() {
		grpcGatewayMux := runtime.NewServeMux(
			grpcgw.WithMarshalerOption(grpcgw.MIMEWildcard, &grpcgw.HTTPBodyMarshaler{
				Marshaler: &grpcgw.JSONPb{
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
		)

		dialOptions := []grpc.DialOption{grpc.WithInsecure()}

		opts.registerGateway(grpcGatewayMux, dialOptions)

		listener, err := net.Listen("tcp", fmt.Sprintf("%v:%v", opts.address, opts.port))

		if err != nil {
			opts.logger.Fatal("API server gateway listener failed to start", zap.Error(err))
		}

		corsHandler := cors.New(cors.Options{
			AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			ExposedHeaders:   []string{"Link", "X-Total-Count"},
			AllowCredentials: true,
		})

		r := chi.NewRouter()
		r.Use(opts.httpMiddleware...)
		r.Use(corsHandler.Handler)
		r.Get("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
		r.Mount("/", grpcGatewayMux)

		server.GrpcGatewayServer = &http.Server{
			Handler: r,
		}

		err = server.GrpcGatewayServer.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			opts.logger.Fatal("API server gateway listener failed", zap.Error(err))
		}
	}()

	return server
}

func (s *Server) Stop() {
	// 1. Stop GRPC Gateway server first as it sits above GRPC server. This also closes the underlying listener.
	if err := s.GrpcGatewayServer.Shutdown(context.Background()); err != nil {
		s.opts.logger.Error("API server gateway listener shutdown failed", zap.Error(err))
	}
	// 2. Stop GRPC server. This also closes the underlying listener.
	s.grpcServer.GracefulStop()
}
