package grpc

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewClientConn creates a client connection to the given addr.
func NewClientConn(
	addr string,
	opt ...OptionClientConn,
) (
	_ *grpc.ClientConn,
	_ error,
) {
	opts := defaultClientConnOptions()

	for _, o := range opt {
		o.apply(opts)
	}

	conn, err := grpc.Dial(addr, opts.dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}

	return conn, nil
}

type clientConnOptions struct {
	dialOptions []grpc.DialOption
}

func defaultClientConnOptions() *clientConnOptions {
	return &clientConnOptions{
		dialOptions: []grpc.DialOption{},
	}
}

// OptionClientConn configures how we set up the connection.
type OptionClientConn interface {
	apply(*clientConnOptions)
}

type funcClientConnOption struct {
	f func(*clientConnOptions)
}

func (f *funcClientConnOption) apply(do *clientConnOptions) {
	f.f(do)
}

func newFuncClientConnOption(f func(*clientConnOptions)) *funcClientConnOption {
	return &funcClientConnOption{
		f: f,
	}
}

// WithNoTLS disables transport security for the client.
// Replacement for grpc.WithInsecure().
func WithNoTLS() OptionClientConn {
	return newFuncClientConnOption(func(o *clientConnOptions) {
		o.dialOptions = append(
			o.dialOptions,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	})
}

// WithContextDialer wraps the grpc.WithContextDialer option.
func WithContextDialer(
	d func(context.Context, string) (net.Conn, error),
) OptionClientConn {
	return newFuncClientConnOption(func(o *clientConnOptions) {
		o.dialOptions = append(
			o.dialOptions,
			grpc.WithContextDialer(d),
		)
	})
}
