package grpc_test

import (
	"context"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/matryer/is"
	commonsgrpc "github.com/purposeinplay/go-commons/grpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func Test(t *testing.T) {
	i := is.New(t)

	grpcServer, err := commonsgrpc.NewServer(
		commonsgrpc.WithDebugStandardLibraryEndpoints(),
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
	i := is.New(t)

	const bufSize = 1024 * 1024

	lis := bufconn.Listen(bufSize)
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	grpcServer, err := commonsgrpc.NewServer(
		commonsgrpc.WithGRPCListener(lis),
		commonsgrpc.WithDebug(zap.NewExample()),
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

	_, err = grpc.Dial(
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	i.NoErr(err)
}
