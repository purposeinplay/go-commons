package grpcclient

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewConn creates a client connection to the given addr.
func NewConn(
	addr string,
	opt ...OptionConn,
) (
	*grpc.ClientConn,
	error,
) {
	opts := defaultClientConnOptions()

	for _, o := range opt {
		o.apply(opts)
	}

	conn, err := grpc.Dial(addr, opts.computeDialOptions()...)
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}

	return conn, nil
}

type connOptions struct {
	dialOptions  []grpc.DialOption
	interceptors []grpc.UnaryClientInterceptor
}

func (o connOptions) computeDialOptions() []grpc.DialOption {
	return append(o.dialOptions, grpc.WithChainUnaryInterceptor(o.interceptors...))
}

func defaultClientConnOptions() *connOptions {
	return &connOptions{
		dialOptions:  []grpc.DialOption{},
		interceptors: []grpc.UnaryClientInterceptor{},
	}
}

// OptionConn configures how we set up the connection.
type OptionConn interface {
	apply(*connOptions)
}

type funcConnOption struct {
	f func(*connOptions)
}

func (f *funcConnOption) apply(do *connOptions) {
	f.f(do)
}

func newFuncConnOption(f func(*connOptions)) *funcConnOption {
	return &funcConnOption{
		f: f,
	}
}

// WithNoTLS disables transport security for the client.
// Replacement for grpc.WithInsecure().
func WithNoTLS() OptionConn {
	return newFuncConnOption(func(o *connOptions) {
		o.dialOptions = append(
			o.dialOptions,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	})
}

// WithContextDialer wraps the grpc.WithContextDialer option.
func WithContextDialer(
	d func(context.Context, string) (net.Conn, error),
) OptionConn {
	return newFuncConnOption(func(o *connOptions) {
		o.dialOptions = append(
			o.dialOptions,
			grpc.WithContextDialer(d),
		)
	})
}

// WithClientUnaryInterceptor adds an interceptor for client calls.
func WithClientUnaryInterceptor(interceptor grpc.UnaryClientInterceptor) OptionConn {
	return newFuncConnOption(func(o *connOptions) {
		o.interceptors = append(
			o.interceptors,
			interceptor,
		)
	})
}
