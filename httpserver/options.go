package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"time"
)

// An Option configures an App using the functional options paradigm
// popularized by Rob Pike. If you're unfamiliar with this style, see
// nolint: revive // line too long.
// https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html
// and
// https://github.com/uber-go/guide/blob/master/style.md#functional-options
type Option interface {
	fmt.Stringer

	apply(*Server)
}

type addressOption string

func (o addressOption) apply(s *Server) {
	s.httpServer.Addr = string(o)
	s.info.Addr = string(o)
}

func (o addressOption) String() string {
	return "server.Address: " + string(o)
}

// WithAddress will set the address field of the server.
func WithAddress(address string) Option {
	return addressOption(address)
}

type serverTimeoutsOption struct {
	fmt.Stringer

	// writeTimeout: the maximum duration before timing out
	// writes of the response
	writeTimeout,
	// readTimeout: the maximum duration for reading
	// the entire request, including the body
	readTimeout,
	// idleTimeout: the maximum amount of time to wait for the next
	// request when keep-alive is enabled
	idleTimeout,
	// readHeaderTimeout: the amount of time allowed to read request headers
	readHeaderTimeout time.Duration
}

func (o serverTimeoutsOption) String() string {
	return fmt.Sprintf("server.WriteTimeout: %d\n"+
		"server.ReadTimeout: %d\n"+
		"server.IdleTimeout: %d\n"+
		"server.ReadHeaderTimeout: %d\n",
		o.writeTimeout,
		o.readTimeout,
		o.idleTimeout,
		o.readHeaderTimeout)
}

func (o serverTimeoutsOption) apply(s *Server) {
	s.httpServer.WriteTimeout = o.writeTimeout
	s.httpServer.ReadTimeout = o.readTimeout
	s.httpServer.IdleTimeout = o.idleTimeout
	s.httpServer.ReadHeaderTimeout = o.readHeaderTimeout
}

// WithServerTimeouts will set the timeouts for the underlying HTTP server.
func WithServerTimeouts(
	writeTimeout,
	readTimeout,
	idleTimeout,
	readHeaderTimeout time.Duration,
) Option {
	return serverTimeoutsOption{
		writeTimeout:      writeTimeout,
		readTimeout:       readTimeout,
		idleTimeout:       idleTimeout,
		readHeaderTimeout: readHeaderTimeout,
	}
}

// nolint: containedctx // allow struct containing ctx as it's an option.
type baseContextOption struct {
	ctx                     context.Context
	cancelContextOnShutdown bool
}

func (o baseContextOption) apply(server *Server) {
	var ctx context.Context

	if o.ctx == nil {
		if !o.cancelContextOnShutdown {
			return
		}

		ctx = context.Background()
	}

	if o.cancelContextOnShutdown {
		var cancel func()
		ctx, cancel = context.WithCancel(o.ctx)

		server.httpServer.RegisterOnShutdown(func() {
			cancel()
		})
	}

	server.httpServer.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}
}

func (o baseContextOption) String() string {
	return fmt.Sprintf(
		"server.BaseContext: %+v\n"+
			"server.CancelContextOnShutdown: %t",
		o.ctx,
		o.cancelContextOnShutdown,
	)
}

// WithBaseContext sets a predefined base context for all
// incoming http requests.
//
// If cancelOnShutdown is set to true it will mark the baseContext
// as done(close the Done channel) whenever
// the server.Shutdown() method is called.
//
// This is intended to use with long living tcp connections like
// Websockets in order to cancel the current open connections and allow
// the server to shutdown.
func WithBaseContext(ctx context.Context, cancelContextOnShutdown bool) Option {
	return baseContextOption{
		ctx,
		cancelContextOnShutdown,
	}
}

type shutdownSignalsOption []os.Signal

func (o shutdownSignalsOption) String() string {
	signals := make([]string, 0, len(o))

	for _, s := range o {
		signals = append(signals, s.String())
	}

	return fmt.Sprintf("server.ShutdownSignals: %s", signals)
}

const shutdownTimeout = 30 * time.Second

func (o shutdownSignalsOption) apply(server *Server) {
	if len(o) == 0 {
		return
	}

	go func() {
		sigC := make(chan os.Signal, 1)

		signal.Notify(sigC, o...)

		server.log.Debug(
			"waiting for shutdown signals",
			slog.String("signals", o.String()),
		)

		sig := <-sigC

		server.log.Info(
			"received shut down signal",
			slog.String("signal", sig.String()),
		)

		err := server.Shutdown(shutdownTimeout)
		if err != nil {
			server.log.Error(
				"failed to shutdown server in time",
				slog.String("error", err.Error()),
			)
		}
	}()
}

// WithShutdownSignalsOption will attempt to shutdown the server
// when one of the proved os signals
// is sent to the application.
func WithShutdownSignalsOption(signals ...os.Signal) Option {
	return shutdownSignalsOption(signals)
}
