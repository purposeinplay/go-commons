package httpserver

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
)

// An Option configures an App using the functional options paradigm
// popularized by Rob Pike. If you're unfamiliar with this style, see
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
	return fmt.Sprintf("server.Address: %s", string(o))
}

// WithAddress will set the address field of the server
func WithAddress(address string) Option {
	return addressOption(address)
}

type serverTimeoutsOption struct {
	fmt.Stringer

	// writeTimeout: the maximum duration before timing out writes of the response
	writeTimeout,
	// readTimeout: the maximum duration for reading the entire request, including the body
	readTimeout,
	// idleTimeout: the maximum amount of time to wait for the next request when keep-alive is enabled
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

type baseContextOption struct {
	ctx                     context.Context
	cancelContextOnShutdown bool
}

func (o baseContextOption) apply(s *Server) {
	if o.ctx == nil {
		if !o.cancelContextOnShutdown {
			return
		}

		o.ctx = context.Background()
	}

	if o.cancelContextOnShutdown {
		var cancel func()
		o.ctx, cancel = context.WithCancel(o.ctx)

		s.httpServer.RegisterOnShutdown(func() {
			cancel()
		})
	}

	s.httpServer.BaseContext = func(_ net.Listener) context.Context {
		return o.ctx
	}
}

func (o baseContextOption) String() string {
	spew.Config.DisablePointerAddresses = true

	return fmt.Sprintf("server.BaseContext: %s"+
		"server.CancelContextOnShutdown: %t", spew.Sdump(o.ctx), o.cancelContextOnShutdown)
}

// WithBaseContext sets a predefined base context for all incoming http requests.
//
// If cancelOnShutdown is set to true it will mark the baseContext as done(close the Done channel) whenever
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

func (o shutdownSignalsOption) apply(s *Server) {
	if len(o) == 0 {
		return
	}

	go func() {
		sigC := make(chan os.Signal, 1)

		signal.Notify(sigC, o...)

		s.log.Debug("waiting for shutdown signals", zap.Stringer("signals", o))

		sig := <-sigC

		s.log.Info("received shut down signal", zap.String("signal", sig.String()))

		err := s.Shutdown(30 * time.Second)
		if err != nil {
			s.log.Error("failed to shutdown server in time", zap.Error(err))
		}
	}()
}

// WithShutdownSignalsOption will attempt to shutdown the server when one of the proved os signals
// is sent to the application
func WithShutdownSignalsOption(signals ...os.Signal) Option {
	return shutdownSignalsOption(signals)
}
