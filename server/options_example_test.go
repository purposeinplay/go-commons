package server_test

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"github.com/purposeinplay/go-commons/server"
)

func ExampleWithShutdownSignalsOption() {
	opt := server.WithShutdownSignalsOption(
		syscall.SIGINT,
		syscall.SIGTERM)

	fmt.Println(opt)
	// Output: server.ShutdownSignals: [interrupt terminated]
}

func ExampleWithAddress() {
	opt := server.WithAddress(":8080")

	fmt.Println(opt)
	// Output: server.Address: :8080
}

func ExampleWithBaseContext() {
	type key string

	ctx := context.WithValue(context.Background(), key("server"), "example")

	opt := server.WithBaseContext(ctx, true)

	fmt.Println(opt)
	// Output:
	// server.BaseContext: (*context.valueCtx)(context.Background.WithValue(type server_test.key, val example))
	// server.CancelContextOnShutdown: true
}

func ExampleWithServerTimeouts() {
	opt := server.WithServerTimeouts(
		time.Nanosecond,
		2*time.Nanosecond,
		3*time.Nanosecond,
		4*time.Nanosecond,
	)

	fmt.Println(opt)
	// Output:
	// server.WriteTimeout: 1
	// server.ReadTimeout: 2
	// server.IdleTimeout: 3
	// server.ReadHeaderTimeout: 4
}
