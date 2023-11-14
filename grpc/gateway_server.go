package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/plugin/ochttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type gatewayServer struct {
	httpServer *http.Server
	listener   net.Listener
	closed     atomic.Bool
}

func (s *gatewayServer) addr() string {
	return s.listener.Addr().String()
}

func (s *gatewayServer) listenAndServe() error {
	err := s.httpServer.Serve(s.listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *gatewayServer) close() error {
	if s.closed.Load() {
		return nil
	}

	s.closed.Store(true)

	return s.httpServer.Shutdown(context.Background())
}

// nolint: revive // false-positive, it reports tracing as a control flag.
func newGatewayServer(
	muxOptions []runtime.ServeMuxOption,
	tracing bool,
	registerGateway registerGatewayFunc,
	address string,
	httpRoutes []httpRoute,
	middlewares chi.Middlewares,
	debugStandardLibraryEndpoints bool,
	corsOptions cors.Options,
) (
	*gatewayServer,
	error,
) {
	grpcGatewayMux := runtime.NewServeMux(
		muxOptions...,
	)

	var handler http.Handler = grpcGatewayMux

	if tracing {
		handler = &ochttp.Handler{
			Handler:     grpcGatewayMux,
			Propagation: &propagation.HTTPFormat{},
		}
	}

	if registerGateway != nil {
		dialOptions := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		}

		err := registerGateway(grpcGatewayMux, dialOptions)
		if err != nil {
			return nil, fmt.Errorf("register gateway: %w", err)
		}
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("new listener: %w", err)
	}

	corsHandler := cors.New(corsOptions)

	router := chi.NewRouter()

	router.Use(middlewares...)
	router.Use(corsHandler.Handler)
	router.Get(
		"/",
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)

	router.Mount("/", handler)

	for _, route := range httpRoutes {
		router.MethodFunc(
			route.method,
			route.path,
			route.handler,
		)
	}

	if debugStandardLibraryEndpoints {
		// Register all the standard library debug endpoints.
		router.Mount("/debug/", middleware.Profiler())
	}

	const (
		handlerTimeout    = 10 * time.Second
		readHeaderTimeout = 5 * time.Second
		wiggleRoom        = 200 * time.Millisecond
		readTimeout       = handlerTimeout + readHeaderTimeout + wiggleRoom
		writeTimeout      = handlerTimeout + wiggleRoom

		idleTimeout = 2 * time.Minute
	)

	return &gatewayServer{
			httpServer: &http.Server{
				Handler: http.TimeoutHandler(
					router,
					handlerTimeout,
					"",
				),
				ReadTimeout:       readTimeout,
				ReadHeaderTimeout: readHeaderTimeout,
				WriteTimeout:      writeTimeout,
				IdleTimeout:       idleTimeout,
			},
			listener: listener,
		},
		nil
}
