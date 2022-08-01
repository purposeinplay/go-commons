package grpc

import (
	"context"
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/purposeinplay/go-commons/grpc/grpcutils"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var _ server = (*grpcServer)(nil)

type grpcServer struct {
	internalGRPCServer *grpc.Server
}

func (s *grpcServer) Serve(listener net.Listener) error {
	return s.internalGRPCServer.Serve(listener)
}

func (s *grpcServer) Close() error {
	s.internalGRPCServer.GracefulStop()

	return nil
}

func newGRPCServerWithListener(
	listener net.Listener,
	address string,
	tracing bool,
	defaultGRPCServerOptions []grpc.ServerOption,
	unaryServerInterceptors []grpc.UnaryServerInterceptor,
	registerServer registerServerFunc,
	debugLogger debugLogger,
) (
	*serverWithListener,
	error,
) {
	grpcListener, err := newGRPCListener(listener, address)
	if err != nil {
		return nil, fmt.Errorf("new grpc listener: %w", err)
	}

	grpcServerOptions, err := setGRPCTracing(tracing, defaultGRPCServerOptions)
	if err != nil {
		return nil, fmt.Errorf("set grpc tracing tracing: %w", err)
	}

	if !isDebugLoggerNil(debugLogger) {
		// nolint: revive // complains that this lines modifies
		// an input parameter.
		unaryServerInterceptors = prependDebugInterceptor(
			unaryServerInterceptors,
			debugLogger,
		)
	}

	if len(unaryServerInterceptors) > 0 {
		grpcServerOptions = append(grpcServerOptions,
			grpc_middleware.WithUnaryServerChain(
				unaryServerInterceptors...,
			))
	}

	internalGRPCServer := grpc.NewServer(grpcServerOptions...)

	reflection.Register(internalGRPCServer)

	if registerServer != nil {
		registerServer(internalGRPCServer)
	}

	return &serverWithListener{
		server: &grpcServer{
			internalGRPCServer: internalGRPCServer,
		},
		listener: grpcListener,
	}, nil
}

// nolint: revive // false-positive, it reports tracing as a control flag.
func setGRPCTracing(
	tracing bool,
	serverOptions []grpc.ServerOption,
) ([]grpc.ServerOption, error) {
	if !tracing {
		return serverOptions, nil
	}

	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: os.Getenv("GOOGLE_CLOUD_PROJECT"),
	})
	if err != nil {
		return nil, fmt.Errorf("new exporter: %w", err)
	}

	trace.RegisterExporter(exporter)
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	return append(
		serverOptions,
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
	), nil
}

func newGRPCListener(
	defaultListener net.Listener,
	addr string,
) (net.Listener, error) {
	if defaultListener != nil {
		return defaultListener, nil
	}

	hostString, portString, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	port, err := strconv.Atoi(portString)
	if err != nil {
		return nil, fmt.Errorf("parse port: %w", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", hostString, port-1))
	if err != nil {
		return nil, fmt.Errorf("new net listener: %w", err)
	}

	return listener, nil
}

func prependDebugInterceptor(
	interceptors []grpc.UnaryServerInterceptor,
	logger debugLogger,
) []grpc.UnaryServerInterceptor {
	return prependServerOption(
		func(
			ctx context.Context,
			req interface{},
			info *grpc.UnaryServerInfo,
			handler grpc.UnaryHandler,
		) (resp interface{}, err error) {
			method := path.Base(info.FullMethod)

			if method == "Check" || method == "Watch" {
				return handler(ctx, req)
			}

			requestID, err := grpcutils.GetRequestIDFromCtx(ctx)
			if err != nil {
				requestID = "00000000-0000-0000-0000-000000000000"
			}

			start := time.Now()

			logger.Debug(
				"request started",
				zap.String("trace_id", requestID),
				zap.String("method", method),
			)

			defer func() {
				logger.Debug(
					"request completed",
					zap.String("trace_id", requestID),
					zap.String("method", method),
					zap.Duration("duration", time.Since(start)),
				)
			}()

			return handler(ctx, req)
		},
		interceptors,
	)
}
