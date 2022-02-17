package grpc

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/rs/cors"

	"github.com/go-chi/chi"

	"go.uber.org/zap"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpcServer        *grpc.Server
	GrpcGatewayServer *http.Server
	opts              serverOptions
}

func NewServer(opt ...ServerOption) *Server {
	opts, err := defaultServerOptions()

	if err != nil {
		opts.logger.Fatal("could not get server options", zap.Error(err))
	}

	for _, o := range opt {
		o.apply(&opts)
	}

	grpcServer := grpc.NewServer(opts.grpcServerOptions...)

	server := &Server{
		opts:       opts,
		grpcServer: grpcServer,
	}

	reflection.Register(grpcServer)

	if opts.registerServer != nil {
		opts.registerServer(grpcServer)
	}

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

	opts.logger.Info(
		"Starting gRPC gateway for HTTP requests",
		zap.String("gRPC gateway", dialAddr),
	)

	waitGatewayInit := make(chan struct{})

	go func() {
		grpcGatewayMux := runtime.NewServeMux(
			opts.muxOptions...,
		)

		if opts.registerGateway != nil {
			dialOptions := []grpc.DialOption{grpc.WithInsecure()}

			opts.registerGateway(grpcGatewayMux, dialOptions)
		}

		listener, err := net.Listen("tcp", fmt.Sprintf("%v:%v", opts.address, opts.port))
		if err != nil {
			opts.logger.Fatal("API server gateway listener failed to start", zap.Error(err))
		}

		if opts.listener != nil {
			listener = opts.listener
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

		close(waitGatewayInit)

		err = server.GrpcGatewayServer.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			opts.logger.Fatal("API server gateway listener failed", zap.Error(err))
		}
	}()

	<-waitGatewayInit

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
