package grpc

import (
	"google.golang.org/grpc"
)

func prependServerOption(
	newInterceptor grpc.UnaryServerInterceptor,
	interceptors []grpc.UnaryServerInterceptor,
) []grpc.UnaryServerInterceptor {
	newInterceptors := make(
		[]grpc.UnaryServerInterceptor,
		len(interceptors)+1,
	)

	copy(newInterceptors[1:], interceptors)

	newInterceptors[0] = newInterceptor

	return newInterceptors
}
