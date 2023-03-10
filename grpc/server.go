package grpc

import (
	"errors"
	"fmt"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/oklog/run"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// ErrServerClosed indicates that the operation is now illegal because of
// the server has been closed.
var ErrServerClosed = errors.New("go-commons.grpc: server closed")

type (
	// registerServerFunc defines how we can register
	// a grpc service to a grpc server.
	registerServerFunc func(server *grpc.Server)

	// registerGatewayFunc defines how we can register
	// a grpc service to a gateway server.
	registerGatewayFunc func(
		mux *runtime.ServeMux,
		dialOptions []grpc.DialOption,
	) error
)

// Server holds the grpc and gateway underlying servers.
// It starts and stops both of them together.
// In case one of the server fails the other one is closed.
type Server struct {
	grpcServer    *grpcServer
	gatewayServer *gatewayServer

	logging *logging

	mu     sync.Mutex
	closed bool
}

// NewServer creates Server with both the grpc server and,
// if it's the case, the gateway server.
//
// The servers have not started to accept requests yet.
func NewServer(opt ...ServerOption) (*Server, error) {
	opts := defaultServerOptions()

	for _, o := range opt {
		o.apply(&opts)
	}

	aggregatorServer := new(Server)

	if opts.logging != nil {
		aggregatorServer.logging = opts.logging
	}

	grpcServerWithListener, err := newGRPCServer(
		opts.grpcListener,
		opts.address,
		opts.tracing,
		opts.grpcServerOptions,
		opts.unaryServerInterceptors,
		opts.registerServer,
		aggregatorServer.logging,
		opts.errorHandler,
		opts.panicHandler,
		opts.monitorOperationer,
	)
	if err != nil {
		return nil, fmt.Errorf("new gRPC server: %w", err)
	}

	aggregatorServer.grpcServer = grpcServerWithListener

	// return here if a gateway server is not wanted.
	if !opts.gateway {
		return aggregatorServer, nil
	}

	grpcGatewayServer, err := newGatewayServer(
		opts.muxOptions,
		opts.tracing,
		opts.registerGateway,
		opts.address,
		opts.httpMiddlewares,
		opts.debugStandardLibraryEndpoints,
		opts.gatewayCorsOptions,
	)
	if err != nil {
		return nil, fmt.Errorf("new gRPC gateway server: %w", err)
	}

	aggregatorServer.gatewayServer = grpcGatewayServer

	return aggregatorServer, nil
}

// ListenAndServe starts accepting incoming connections on both servers.
// If one of the servers encounters an error, both are stopped.
func (s *Server) ListenAndServe() error {
	s.mu.Lock()

	if s.closed {
		s.mu.Unlock()

		return ErrServerClosed
	}

	var runGroup run.Group

	runGroup.Add(s.runGRPCServer, func(err error) {
		_ = s.grpcServer.close()
	})

	// start gateway server.
	if s.gatewayServer != nil {
		runGroup.Add(s.runGatewayServer, func(err error) {
			_ = s.gatewayServer.close()
		})
	}

	s.mu.Unlock()

	err := runGroup.Run()
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) runGRPCServer() error {
	s.logDebug(
		"starting gRPC server",
		zap.String(
			"address",
			s.grpcServer.addr(),
		),
	)

	return s.grpcServer.listenAndServe()
}

func (s *Server) closeGRPCServer() error {
	s.logDebug("close grpc server")
	defer s.logDebug("grpc server done")

	return s.grpcServer.close()
}

func (s *Server) runGatewayServer() error {
	s.logDebug(
		"starting gRPC gateway server for HTTP requests",
		zap.String(
			"address",
			s.gatewayServer.addr(),
		),
	)

	return s.gatewayServer.listenAndServe()
}

func (s *Server) closeGatewayServer() error {
	s.logDebug("close gateway server")
	defer s.logDebug("gateway server done")

	return s.gatewayServer.close()
}

// Close closes both underlying servers.
// Safe to use concurrently and can be called multiple times.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	// 1. Stop GRPC Gateway server first as it sits above GRPC server.
	// This also closes the underlying grpcListener.
	if s.gatewayServer != nil {
		err := s.closeGatewayServer()
		if err != nil {
			return fmt.Errorf("close gateway server: %w", err)
		}
	}

	// 2. Stop GRPC server. This also closes the underlying grpcListener.
	err := s.closeGRPCServer()
	if err != nil {
		return fmt.Errorf("close grpc server: %w", err)
	}

	return nil
}

func (s *Server) logDebug(msg string, fields ...zap.Field) {
	if s.logging == nil {
		return
	}

	s.logging.logger.Debug(msg, fields...)
}
