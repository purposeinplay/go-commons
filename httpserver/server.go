package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	defaultAddr = ":80"
)

// Info holds relevant information about the Server.
// This can be used in the future to hold information about:
// - number of requests received
// - average response time.
type Info struct {
	Addr string
}

// Server handles the setup and shutdown of the http server
// for an http.Handler.
type Server struct {
	// underlying http server
	httpServer *http.Server

	log *slog.Logger

	// chan to signal that the server was shutdown which means that either the
	// Server() or ListenAndServe() methods returned.
	done chan struct{}

	// holds extra information about the service
	info Info

	// once function to only close the done channel once.
	closeDoneOnce sync.Once
}

// New will build a server with the defaults in place.
// You can use Options to override the defaults.
// Default list:
// - Address: ":8080".
func New(logger *slog.Logger, handler http.Handler, options ...Option) *Server {
	const (
		handlerTimeout    = 120 * time.Second
		readHeaderTimeout = 5 * time.Second
		wiggleRoom        = 200 * time.Millisecond
		readTimeout       = handlerTimeout + readHeaderTimeout + wiggleRoom
		writeTimeout      = handlerTimeout + wiggleRoom

		idleTimeout = 2 * time.Minute
	)

	server := &Server{
		httpServer: &http.Server{
			Addr: defaultAddr,

			Handler: http.TimeoutHandler(
				handler,
				handlerTimeout,
				"",
			),
			ReadTimeout:       readTimeout,
			ReadHeaderTimeout: readHeaderTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		log:  logger,
		done: make(chan struct{}),
	}

	for _, o := range options {
		o.apply(server)
	}

	return server
}

// Shutdown is a wrapper over http.Server.Shutdown() that also closes the
// Server done channel and sets a timeout for the shutdown operation.
func (s *Server) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	defer s.closeDoneOnce.Do(func() {
		close(s.done)
	})

	err := s.httpServer.Shutdown(ctx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Serve is a wrapper over http.Server.Serve(), and accepts incoming connections
// on the provided listener.
func (s *Server) Serve(ln net.Listener) error {
	err := s.httpServer.Serve(ln)

	err = s.handleShutdown(err)
	if err != nil {
		return err
	}

	return nil
}

// ListenAndServe is a wrapper over http.Server.ListenAndServe()
// that logs basic information
// and blocks execution until the Server.Shutdown() method is called.
func (s *Server) ListenAndServe() error {
	s.log.Info("starting server", slog.String("address", s.httpServer.Addr))

	err := s.httpServer.ListenAndServe()

	err = s.handleShutdown(err)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) handleShutdown(err error) error {
	// log that the server shutdown
	s.log.Debug("listener shutdown, waiting for connections to drain")

	// wait until Shutdown() method returns
	<-s.done

	s.log.Debug("server connections are drained")

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Info returns the server.Info object.
func (s *Server) Info() Info {
	return s.info
}
