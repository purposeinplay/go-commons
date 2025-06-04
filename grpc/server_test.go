package grpc_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/matryer/is"
	commonsgrpc "github.com/purposeinplay/go-commons/grpc"
	"github.com/purposeinplay/go-commons/grpc/grpcclient"
	"github.com/purposeinplay/go-commons/grpc/test_data/greetpb"
	"github.com/purposeinplay/go-commons/grpc/test_data/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestGateway(t *testing.T) {
	t.Skip()
	const body = `{"greeting":{"first_name":"John","last_name":"Doe"}}`

	t.Run("Success", func(t *testing.T) {
		i := is.New(t)

		header := "test"

		grpcServer, err := commonsgrpc.NewServer(
			commonsgrpc.WithDebug(slog.Default(), true),
			commonsgrpc.WithMuxOptions([]runtime.ServeMuxOption{
				runtime.WithErrorHandler(func(
					_ context.Context,
					_ *runtime.ServeMux,
					_ runtime.Marshaler,
					respWriter http.ResponseWriter,
					_ *http.Request,
					err error,
				) {
					t.Logf("err: %s", err)

					// return Internal when Marshal failed
					const fallback = `{"code": 13, "message": "failed to marshal error message"}`

					respWriter.Header().Set("Content-Type", "application/json")
					respWriter.Write([]byte(fallback))
				}),
			}),
			commonsgrpc.WithHTTPMiddlewares(
				chi.Middlewares{func(handler http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if !strings.Contains(r.URL.Path, "greet") {
							handler.ServeHTTP(w, r)
							return
						}

						b, err := io.ReadAll(r.Body)
						if err != nil {
							i.NoErr(err)
						}

						r.Body = io.NopCloser(bytes.NewReader(b))

						t.Logf(
							"middleware body: %s, custom header: %s",
							string(b),
							r.Header.Get("X-Custom"),
						)

						i.Equal(body, string(b))
						i.Equal(header, r.Header.Get("X-Custom"))

						handler.ServeHTTP(w, r)
					})
				}}),
			commonsgrpc.WithHTTPRoute(
				http.MethodGet,
				"/test",
				func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusUnprocessableEntity)
				},
			),
			commonsgrpc.WithRegisterServerFunc(func(server *grpc.Server) {
				greetpb.RegisterGreetServiceServer(server, &greeterService{
					greetFunc: func() error { return nil },
				})
			}),
			commonsgrpc.WithRegisterGatewayFunc(func(
				mux *runtime.ServeMux,
				dialOptions []grpc.DialOption,
			) error {
				err := greetpb.RegisterGreetServiceHandlerFromEndpoint(
					context.Background(),
					mux,
					"0.0.0.0:7349",
					dialOptions,
				)
				if err != nil {
					return fmt.Errorf("register gRPC gateway: %w", err)
				}

				return nil
			}),
		)
		i.NoErr(err)

		t.Logf("starting server")

		go func() {
			err := grpcServer.ListenAndServe()
			if err != nil {
				panic(err)
			}
		}()

		t.Cleanup(func() {
			err := grpcServer.Close()
			if err != nil {
				panic(err)
			}
		})

		t.Logf("test request")

		resp, err := http.DefaultClient.Get("http://0.0.0.0:7350/test")
		i.NoErr(err)

		err = resp.Body.Close()
		i.NoErr(err)

		i.Equal(resp.StatusCode, http.StatusUnprocessableEntity)

		req, err := http.NewRequest(
			http.MethodPost,
			"http://0.0.0.0:7350/greet",
			strings.NewReader(body),
		)
		i.NoErr(err)

		req.Header.Set("Grpc-Metadata-Custom", header)
		req.Header.Set("X-Custom", header)

		t.Logf("sending request")

		resp, err = http.DefaultClient.Do(req)
		i.NoErr(err)

		b, err := io.ReadAll(resp.Body)
		i.NoErr(err)

		i.NoErr(resp.Body.Close())

		i.Equal(string(b), `{"result":"JohnDoetest"}`)
	})

	t.Run("Error", func(t *testing.T) {
		i := is.New(t)

		// nolint: goerr113
		grpcErr := errors.New("custom test error")

		const errorMessage = `{"code": 13, "message": "failed to marshal error message"}`

		grpcServer, err := commonsgrpc.NewServer(
			commonsgrpc.WithMuxOptions([]runtime.ServeMuxOption{
				runtime.WithErrorHandler(func(
					_ context.Context,
					_ *runtime.ServeMux,
					_ runtime.Marshaler,
					w http.ResponseWriter,
					_ *http.Request,
					err error,
				) {
					i.Equal(grpcErr.Error(), status.Convert(err).Message())

					w.Write([]byte(errorMessage))
				}),
			}),
			commonsgrpc.WithRegisterServerFunc(func(server *grpc.Server) {
				greetpb.RegisterGreetServiceServer(server, &greeterService{
					greetFunc: func() error { return grpcErr },
				})
			}),
			commonsgrpc.WithRegisterGatewayFunc(func(
				mux *runtime.ServeMux,
				dialOptions []grpc.DialOption,
			) error {
				err := greetpb.RegisterGreetServiceHandlerFromEndpoint(
					context.Background(),
					mux,
					"0.0.0.0:7349",
					dialOptions,
				)
				if err != nil {
					return fmt.Errorf("register gRPC gateway: %w", err)
				}

				return nil
			}),
		)
		i.NoErr(err)

		go func() {
			err := grpcServer.ListenAndServe()
			if err != nil {
				panic(err)
			}
		}()

		t.Cleanup(func() {
			err := grpcServer.Close()
			if err != nil {
				panic(err)
			}
		})

		req, err := http.NewRequest(
			http.MethodPost,
			"http://0.0.0.0:7350/greet",
			strings.NewReader(body),
		)
		i.NoErr(err)

		resp, err := http.DefaultClient.Do(req)
		i.NoErr(err)

		b, err := io.ReadAll(resp.Body)
		i.NoErr(err)

		i.NoErr(resp.Body.Close())

		i.Equal(errorMessage, string(b))
	})
}

func TestPort(t *testing.T) {
	i := is.New(t)

	grpcServer, err := commonsgrpc.NewServer(
		commonsgrpc.WithDebugStandardLibraryEndpoints(),
		commonsgrpc.WithErrorHandler(nil),
	)
	i.NoErr(err)

	go func() {
		err := grpcServer.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	t.Cleanup(func() {
		err := grpcServer.Close()
		if err != nil {
			panic(err)
		}
	})

	resp, err := http.Get("http://localhost:7350/debug/pprof")
	i.NoErr(err)

	err = resp.Body.Close()
	i.NoErr(err)

	i.Equal(http.StatusOK, resp.StatusCode)

	resp, err = http.Get("http://localhost:7350/debug/pprof/allocs")
	i.NoErr(err)

	err = resp.Body.Close()
	i.NoErr(err)

	i.Equal(http.StatusOK, resp.StatusCode)

	err = grpcServer.Close()
	i.NoErr(err)
}

func TestBufnet(t *testing.T) {
	t.Parallel()

	i := is.New(t)

	const bufSize = 1024 * 1024

	lis := bufconn.Listen(bufSize)
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	grpcServer, err := commonsgrpc.NewServer(
		commonsgrpc.WithGRPCListener(lis),
		commonsgrpc.WithDebug(slog.Default(), true),
	)
	i.NoErr(err)

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		t.Log("listen and serve")
		defer t.Log("listen and serve done")

		err := grpcServer.ListenAndServe()
		i.NoErr(err)
	}()

	time.Sleep(time.Second / 10)

	t.Cleanup(func() {
		t.Log("close")

		err := grpcServer.Close()
		i.NoErr(err)

		t.Log("close called")

		wg.Wait()
	})

	_, err = grpc.NewClient(
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	i.NoErr(err)
}

func TestErrorHandling(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// nolint: goerr113 // allow dynamic error for this sentinel error.
	appErr := errors.New("err")

	const panicString = "panic"

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		errorHandler := &mock.ErrorHandlerMock{
			ErrorToGRPCStatusFunc: func(
				error,
			) (*status.Status, error) {
				return status.New(codes.Internal, appErr.Error()), nil
			},
			IsApplicationErrorFunc: func(err error) bool {
				return errors.Is(err, appErr)
			},
			LogErrorFunc: func(err error) {
				log.Printf("log err: %s", err.Error())
			},
			ReportErrorFunc: func(
				_ context.Context,
				err error,
			) error {
				log.Printf("report err: %s", err.Error())
				return nil
			},
		}

		bufDialer := newBufnetServer(
			t,
			&greeterService{
				greetFunc: func() error {
					return appErr
				},
			},
			errorHandler,
			nil,
			nil,
		)

		greetClient := newGreeterClient(t, "bufnet", bufDialer)

		resp, err := greetClient.Greet(ctx, &greetpb.GreetRequest{
			Greeting: &greetpb.Greeting{
				FirstName: "a",
				LastName:  "b",
			},
		})

		i.Equal(status.Error(codes.Internal, appErr.Error()), err)

		i.True(resp == nil)

		i.Equal(0, len(errorHandler.ReportErrorCalls()))

		i.True(errors.Is(errorHandler.IsApplicationErrorCalls()[0].Err, appErr))
		i.True(errors.Is(errorHandler.ErrorToGRPCStatusCalls()[0].Err, appErr))
		i.True(errors.Is(errorHandler.LogErrorCalls()[0].Err, appErr))
	})

	t.Run("Panic", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		panicHandler := &mock.PanicHandlerMock{
			LogErrorFunc: func(err error) {
				log.Printf("log err: %s", err.Error())
			},
			LogPanicFunc: func(p any) {
				log.Printf("log panic: %s", p)
			},
			ReportPanicFunc: func(_ context.Context, p any) error {
				log.Printf("report panic: %s", p)
				return nil
			},
		}

		bufDialer := newBufnetServer(
			t,
			&greeterService{
				greetFunc: func() error {
					panic(panicString)
				},
			},
			nil,
			panicHandler,
			nil,
		)

		greetClient := newGreeterClient(t, "bufnet", bufDialer)

		resp, err := greetClient.Greet(ctx, &greetpb.GreetRequest{
			Greeting: &greetpb.Greeting{
				FirstName: "a",
				LastName:  "b",
			},
		})
		i.Equal(status.Error(codes.Internal, "internal error."), err)

		i.True(resp == nil)

		i.Equal(panicString, panicHandler.LogPanicCalls()[0].IfaceVal)
		i.Equal(panicString, panicHandler.ReportPanicCalls()[0].IfaceVal)
	})

	t.Run("ErrorAndPanic", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		errorHandler := &mock.ErrorHandlerMock{
			IsApplicationErrorFunc: func(error) bool {
				panic(panicString)
			},
			LogErrorFunc: func(err error) {
				log.Printf("log err: %s", err.Error())
			},
		}

		panicHandler := &mock.PanicHandlerMock{
			LogErrorFunc: func(err error) {
				log.Printf("log err: %s", err.Error())
			},
			LogPanicFunc: func(p any) {
				log.Printf("log panic: %s", p)
			},
			ReportPanicFunc: func(_ context.Context, p any) error {
				log.Printf("report panic: %s", p)
				return nil
			},
		}

		bufDialer := newBufnetServer(
			t,
			&greeterService{
				greetFunc: func() error {
					return appErr
				},
			},
			errorHandler,
			panicHandler,
			nil,
		)

		greetClient := newGreeterClient(t, "bufnet", bufDialer)

		resp, err := greetClient.Greet(ctx, &greetpb.GreetRequest{
			Greeting: &greetpb.Greeting{
				FirstName: "a",
				LastName:  "b",
			},
		})
		i.Equal(status.Error(codes.Internal, "internal error."), err)

		i.True(resp == nil)

		i.Equal(0, len(errorHandler.ReportErrorCalls()))
		i.Equal(0, len(errorHandler.ErrorToGRPCStatusCalls()))

		i.True(errors.Is(errorHandler.IsApplicationErrorCalls()[0].Err, appErr))
		i.True(errors.Is(errorHandler.LogErrorCalls()[0].Err, appErr))
		i.Equal(panicString, panicHandler.LogPanicCalls()[0].IfaceVal)
		i.Equal(panicString, panicHandler.ReportPanicCalls()[0].IfaceVal)
	})
}

func TestMonitorOperation(t *testing.T) {
	t.Parallel()

	i := is.New(t)

	ctx := context.Background()

	monitorOperationer := &mock.MonitorOperationerMock{
		MonitorOperationFunc: func(
			ctx context.Context,
			_ string,
			_ [16]byte,
			f func(context.Context),
		) {
			f(ctx)
		},
	}

	bufDialer := newBufnetServer(
		t,
		&greeterService{},
		nil,
		nil,
		monitorOperationer,
	)

	greetClient := newGreeterClient(t, "bufnet", bufDialer)

	resp, err := greetClient.Greet(ctx, &greetpb.GreetRequest{
		Greeting: &greetpb.Greeting{
			FirstName: "a",
			LastName:  "b",
		},
	})
	i.NoErr(err)

	i.Equal("ab", resp.Result)

	i.Equal(1, len(monitorOperationer.MonitorOperationCalls()))
}

var _ greetpb.GreetServiceServer = (*greeterService)(nil)

type greeterService struct {
	greetpb.UnimplementedGreetServiceServer
	greetFunc func() error
}

func (s *greeterService) Greet(
	ctx context.Context,
	req *greetpb.GreetRequest,
) (*greetpb.GreetResponse, error) {
	if s.greetFunc != nil {
		err := s.greetFunc()
		if err != nil {
			return nil, err
		}
	}

	res := req.Greeting.FirstName + req.Greeting.LastName

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if len(md["custom"]) > 0 {
			res += md["custom"][0]
		}
	}

	return &greetpb.GreetResponse{
		Result: res,
	}, nil
}

// nolint: contextcheck // no need to pass context here.
func newBufnetServer(
	t *testing.T,
	greeter *greeterService,
	errorHandler *mock.ErrorHandlerMock,
	panicHandler *mock.PanicHandlerMock,
	monitorOperationer *mock.MonitorOperationerMock,
) func(context.Context, string) (net.Conn, error) {
	t.Helper()

	i := is.New(t)

	const bufSize = 1024 * 1024

	lis := bufconn.Listen(bufSize)

	t.Cleanup(func() {
		err := lis.Close()
		if err != nil {
			t.Logf("err while closing listener: %s", err)
		}
	})

	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	opts := []commonsgrpc.ServerOption{
		commonsgrpc.WithGRPCListener(lis),
		commonsgrpc.WithDebug(slog.Default(), true),
	}

	if greeter != nil {
		opts = append(
			opts,
			commonsgrpc.WithRegisterServerFunc(func(server *grpc.Server) {
				greetpb.RegisterGreetServiceServer(server, greeter)
			}))
	}

	if panicHandler != nil {
		opts = append(
			opts,
			commonsgrpc.WithPanicHandler(panicHandler),
		)
	}

	if errorHandler != nil {
		opts = append(
			opts,
			commonsgrpc.WithErrorHandler(errorHandler),
		)
	}

	if monitorOperationer != nil {
		opts = append(
			opts,
			commonsgrpc.WithMonitorOperationer(monitorOperationer),
		)
	}

	grpcServer, err := commonsgrpc.NewServer(opts...)
	i.NoErr(err)

	var wg sync.WaitGroup

	t.Cleanup(func() {
		wg.Wait()
	})

	wg.Add(1)

	go func() {
		defer wg.Done()

		t.Log("listen and serve")
		defer t.Log("listen and serve done")

		err := grpcServer.ListenAndServe()
		i.NoErr(err)
	}()

	t.Cleanup(func() {
		err := grpcServer.Close()
		if err != nil {
			t.Logf("err while closing server: %s", err)
		}
	})

	return bufDialer
}

func newGreeterClient(
	t *testing.T,
	addr string,
	dialer func(context.Context, string) (net.Conn, error),
) greetpb.GreetServiceClient {
	t.Helper()

	i := is.New(t)

	clientConn, err := grpcclient.NewConn(
		addr,
		grpcclient.WithContextDialer(dialer),
		grpcclient.WithNoTLS(),
	)
	i.NoErr(err)

	t.Cleanup(func() {
		err := clientConn.Close()
		if err != nil {
			t.Logf("err while closing conn: %s", err)
		}
	})

	return greetpb.NewGreetServiceClient(clientConn)
}
