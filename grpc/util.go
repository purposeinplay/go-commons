package grpc

import (
	"reflect"

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

func isDebugLoggerNil(logger debugLogger) bool {
	return logger == nil || reflect.ValueOf(logger).IsNil()
}
