package httpserver_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/purposeinplay/go-commons/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

func TestServer_ShutdownWithoutCallingListenAndServe(t *testing.T) {
	s := httpserver.New(zap.NewExample(), nil)

	err := s.Shutdown(0)
	assert.NoError(t, err)
}

func TestServer_DoubleShutdown(t *testing.T) {
	s := httpserver.New(zap.NewExample(), nil)

	err := s.Shutdown(0)
	require.NoError(t, err)

	err = s.Shutdown(0)
	assert.NoError(t, err)
}

func TestServer(t *testing.T) {
	// exitStatus for the http request handler
	const (
		_ uint32 = iota
		// exit status is set to 1 when the request handler
		// returns due to context getting cancelled
		exitContext

		// exit status is set to 2 when the request handler
		// returns due to the timeout
		exitTimeAfter
	)

	var (
		// wait group in order to sync the:
		// - go routine for http request handler
		// - go routine for the Server
		// - go routine for the http request sender
		wg sync.WaitGroup

		// chan used to signal that the server received
		// the request and can be shut down.
		shutdownServer chan struct{}

		handlerExitStatus atomic.Uint32

		// the default handler used by the server
		defaultHandler = func() http.Handler {
			return http.HandlerFunc(func(
				_ http.ResponseWriter,
				r *http.Request,
			) {
				defer wg.Done()

				// signal that the request is received and
				// the server can be shut down.
				close(shutdownServer)

				// return either by receiving a context done or by a timeout
				select {
				case <-r.Context().Done():
					handlerExitStatus.Store(exitContext)

				case <-time.After(2 * time.Second):
					handlerExitStatus.Store(exitTimeAfter)

				}
			})
		}
	)

	// holds server options
	type serverOptions struct {
		logger  *zap.Logger
		handler http.Handler
		options []httpserver.Option
	}

	tests := map[string]struct {
		// options to the applied to the server
		serverOptions serverOptions

		// shutdown signals to be sent to the program
		shutdownSignals []os.Signal

		// timeout to be used for server.Shutdown()
		shutdownTimeout time.Duration

		// extra values to be added to the waitgroup
		// in case we want to do more stuff in the handler
		extraWgCounter int

		// expected error returned from server.Shutdown()
		expectedShutdownError error

		expectedHandlerExitStatus uint32
	}{
		// server will shutdown after request finishes due to increased timeout
		"ShutdownWithoutClosingLongLivedConnectionContext": {
			serverOptions: serverOptions{
				logger:  zap.NewExample(),
				handler: defaultHandler(),
			},

			extraWgCounter:            1,
			shutdownTimeout:           5 * time.Second,
			expectedHandlerExitStatus: exitTimeAfter,
			expectedShutdownError:     nil,
		},

		// server will return on shutdown before the http
		// request finishes due to the timeout
		"ShutdownDeadlineExceeded": {
			serverOptions: serverOptions{
				logger:  zap.NewExample(),
				handler: defaultHandler(),
			},

			extraWgCounter:            1,
			shutdownTimeout:           time.Second,
			expectedHandlerExitStatus: exitTimeAfter,
			expectedShutdownError:     context.DeadlineExceeded,
		},

		// server shutdown wll also close the request context
		"ShutdownWithClosingBaseContext": {
			serverOptions: serverOptions{
				logger:  zap.NewExample(),
				handler: defaultHandler(),
				options: []httpserver.Option{httpserver.WithBaseContext(
					context.Background(),
					true,
				)},
			},

			extraWgCounter:            1,
			shutdownTimeout:           time.Second,
			expectedHandlerExitStatus: exitContext,
			expectedShutdownError:     nil,
		},

		"ShutdownWithSignals": {
			serverOptions: serverOptions{
				logger:  zap.NewExample(),
				handler: defaultHandler(),
				options: []httpserver.Option{
					httpserver.WithShutdownSignalsOption(syscall.SIGINT),
				},
			},
			extraWgCounter:            1,
			shutdownSignals:           []os.Signal{syscall.SIGINT},
			expectedHandlerExitStatus: exitTimeAfter,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// initialize the shutdown server chan everytime a test is run
			shutdownServer = make(chan struct{})

			// create new server
			s := httpserver.New(
				test.serverOptions.logger,
				test.serverOptions.handler,
				test.serverOptions.options...)

			// add 2 to the waitgroup for the http request and
			// http server go routines.
			// add extra counters given by the test
			wg.Add(2 + test.extraWgCounter)

			// create a new listener for the given addres
			ln, err := net.Listen("tcp", s.Info().Addr)
			require.NoError(t, err)

			go func() {
				defer wg.Done()

				// start accepting requests
				err := s.Serve(ln)
				require.NoError(t, err)

				t.Logf("server complete")
			}()

			address := fmt.Sprintf(
				"http://127.0.0.1:%d",
				ln.Addr().(*net.TCPAddr).Port,
			)

			t.Logf("server listening on: %s", address)

			go func() {
				defer wg.Done()

				// send a request to the server
				resp, err := http.Get(address)
				require.NoError(t, err)

				switch handlerExitStatus.Load() {
				// due to http.TimeoutHandler 503 is returned when
				// request's context is cancelled.
				case exitContext:
					require.Equal(
						t,
						http.StatusServiceUnavailable,
						resp.StatusCode,
					)
				case exitTimeAfter:
					require.Equal(t, http.StatusOK, resp.StatusCode)
				default:
					require.Equal(t, http.StatusOK, resp.StatusCode)
				}

				err = resp.Body.Close()
				assert.NoError(t, err)

				t.Logf("request complete")
			}()

			// wait for the request to be handled and
			// to send the shutdownServer signal
			<-shutdownServer

			if len(test.shutdownSignals) > 0 {
				t.Logf("sending shutdown signals")

				// send the shutdown signals
				err := sendSignals(test.shutdownSignals...)
				require.NoError(t, err)
			} else {
				t.Logf("calling server.Shutdown()")

				// shutdown the server
				err := s.Shutdown(test.shutdownTimeout)
				assert.ErrorIs(t, err, test.expectedShutdownError)
			}

			t.Logf("shutdown complete")

			// wait for the go routines to return
			wg.Wait()

			if test.serverOptions.handler != nil {
				assert.Equal(
					t,
					test.expectedHandlerExitStatus,
					handlerExitStatus.Load(),
				)
			}
		})
	}
}

func sendSignals(signals ...os.Signal) error {
	p, err := os.FindProcess(syscall.Getpid())
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}

	for _, s := range signals {
		err = p.Signal(s)
		if err != nil {
			return fmt.Errorf("send signal: %w", err)
		}
	}

	return nil
}
