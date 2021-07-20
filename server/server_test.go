package server_test

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

	"github.com/purposeinplay/go-commons/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServer_ShutdownWithoutCallingListenAndServe(t *testing.T) {
	s := server.New(zap.NewExample(), nil)

	err := s.Shutdown(0)
	assert.NoError(t, err)
}

func TestServer_DoubleShutdown(t *testing.T) {
	s := server.New(zap.NewExample(), nil)

	err := s.Shutdown(0)
	require.NoError(t, err)

	err = s.Shutdown(0)
	assert.NoError(t, err)
}

func TestServer(t *testing.T) {
	type exitStatus uint8

	const (
		exitDefault exitStatus = iota
		exitContext
		exitTimeAfter
	)

	var (
		wg sync.WaitGroup

		shutdownServer chan struct{}

		handlerExitStatus exitStatus

		defaultHandler = func() http.Handler {
			return http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				defer wg.Done()

				close(shutdownServer)

				select {
				case <-r.Context().Done():
					handlerExitStatus = exitContext

				case <-time.After(2 * time.Second):
					handlerExitStatus = exitTimeAfter
				}
			})
		}
	)

	type serverOptions struct {
		logger  *zap.Logger
		handler http.Handler
		options []server.Option
	}

	tests := map[string]struct {
		serverOptions             serverOptions
		shutdownSignals           []os.Signal
		shutdownTimeout           time.Duration
		extraWgCounter            int
		expectedShutdownError     error
		expectedHandlerExitStatus exitStatus
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

		// server will return on shutdown before the http request finishes due to the timeout
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
				options: []server.Option{server.WithBaseContext(
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
				options: []server.Option{
					server.WithShutdownSignalsOption(syscall.SIGINT),
				},
			},
			extraWgCounter:            1,
			shutdownSignals:           []os.Signal{syscall.SIGINT},
			expectedHandlerExitStatus: exitTimeAfter,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			shutdownServer = make(chan struct{})

			s := server.New(
				test.serverOptions.logger,
				test.serverOptions.handler,
				test.serverOptions.options...)

			wg.Add(2 + test.extraWgCounter)

			ln, err := net.Listen("tcp", s.Info().Addr)
			require.NoError(t, err)

			go func() {
				defer wg.Done()

				err = s.Serve(ln)
				require.NoError(t, err)

				t.Logf("server complete")
			}()

			go func() {
				defer wg.Done()

				resp, err := http.Get("http://127.0.0.1:8080")
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode)

				err = resp.Body.Close()
				assert.NoError(t, err)

				t.Logf("request complete")
			}()

			<-shutdownServer

			if len(test.shutdownSignals) > 0 {
				t.Logf("sending shutdown signals")

				err := sendSignals(test.shutdownSignals...)
				require.NoError(t, err)

			} else {
				t.Logf("calling server.Shutdown()")

				err := s.Shutdown(test.shutdownTimeout)
				assert.ErrorIs(t, err, test.expectedShutdownError)
			}

			t.Logf("shutdown complete")

			wg.Wait()

			if test.serverOptions.handler != nil {
				assert.Equal(t, test.expectedHandlerExitStatus, handlerExitStatus)
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
