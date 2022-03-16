package grpc_test

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/matryer/is"
	commonsgrpc "github.com/purposeinplay/go-commons/grpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func TestBufnet(t *testing.T) {
	i := is.New(t)

	const bufSize = 1024 * 1024

	lis := bufconn.Listen(bufSize)
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	s, err := commonsgrpc.NewServer(
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

		err := s.ListenAndServe()
		i.NoErr(err)
	}()

	time.Sleep(time.Second / 10)

	t.Cleanup(func() {
		t.Log("close")

		err := s.Close()
		i.NoErr(err)

		t.Log("close called")

		wg.Wait()
	})

	_, err = grpc.Dial(
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithInsecure(),
	)
	i.NoErr(err)
}
