package grpc

import (
	"context"
	"fmt"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"net"
	"net/http"
	"os"

	"github.com/rs/cors"

	"github.com/go-chi/chi/v5"

	"go.uber.org/zap"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
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

	if opts.tracing {
		exporter, err := stackdriver.NewExporter(stackdriver.Options{
			ProjectID: os.Getenv("GOOGLE_CLOUD_PROJECT"),
		})
		if err != nil {
			opts.logger.Fatal("could not instantiate exporter", zap.Error(err))
		}
		trace.RegisterExporter(exporter)
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	}

	if opts.tracing {
		opts.grpcServerOptions = append(opts.grpcServerOptions, grpc.StatsHandler(&ocgrpc.ServerHandler{}))
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

	listener := opts.grpcListener

	if listener == nil {
		listener, err = net.Listen("tcp", fmt.Sprintf("%v:%v", opts.address, opts.port-1))
		if err != nil {
			opts.logger.Fatal("gRPC server listener failed to start", zap.Error(err))
		}
	}

	go func() {
		err = grpcServer.Serve(listener)
		if err != nil {
			opts.logger.Fatal("gRPC server listener failed", zap.Error(err))
		}
	}()

	if !opts.gateway {
		return server
	}

	// Register and start GRPC Gateway server.
	dialAddr := fmt.Sprintf("127.0.0.1:%d", opts.port)
	if opts.address != "" {
		dialAddr = fmt.Sprintf("%v:%d", opts.address, opts.port)
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

		var handler http.Handler
		if opts.tracing {
			handler = &ochttp.Handler{
				Handler:     grpcGatewayMux,
				Propagation: &propagation.HTTPFormat{},
			}
		} else {
			handler = grpcGatewayMux
		}

		if opts.registerGateway != nil {
			dialOptions := []grpc.DialOption{
				grpc.WithInsecure(),
				grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
			}

			opts.registerGateway(grpcGatewayMux, dialOptions)
		}

		listener, err := net.Listen("tcp", fmt.Sprintf(
			"%v:%v",
			opts.address,
			opts.port,
		))
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
		r.Mount("/", handler)

		server.GrpcGatewayServer = &http.Server{
			Handler: r,
		}

		close(waitGatewayInit)

		err = server.GrpcGatewayServer.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			opts.logger.Fatal("API server gateway grpcListener failed", zap.Error(err))
		}
	}()

	<-waitGatewayInit

	return server
}

func (s *Server) Stop() {
	// 1. Stop GRPC Gateway server first as it sits above GRPC server. This also closes the underlying grpcListener.
	if s.GrpcGatewayServer != nil {
		if err := s.GrpcGatewayServer.Shutdown(context.Background()); err != nil {
			s.opts.logger.Error("API server gateway grpcListener shutdown failed", zap.Error(err))
		}
	}

	// 2. Stop GRPC server. This also closes the underlying grpcListener.
	s.grpcServer.GracefulStop()
}
